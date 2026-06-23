package service

import (
	"fitness-studio-api/config"
	"fitness-studio-api/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ServiceError struct {
	Status  int
	Message string
	Data    gin.H
}

func (e *ServiceError) Error() string {
	return e.Message
}

func NewServiceError(status int, message string) *ServiceError {
	return &ServiceError{Status: status, Message: message}
}

func NewServiceErrorWithData(status int, message string, data gin.H) *ServiceError {
	return &ServiceError{Status: status, Message: message, Data: data}
}

type BookingResult struct {
	IsWaitlist   bool
	Booking      *models.Booking
	Waitlist     *models.Waitlist
	Schedule     *models.Schedule
	Member       *models.User
	Position     int
	TotalWaiting int
}

func CreateBooking(memberID, scheduleID uint) (*BookingResult, *ServiceError) {
	var member models.User
	if err := models.DB.Where("id = ? AND role = ?", memberID, config.RoleMember).First(&member).Error; err != nil {
		return nil, NewServiceError(404, "会员不存在")
	}

	if member.MembershipStatus() == config.MembershipStatusExpired {
		return nil, NewServiceErrorWithData(400, "会员已过期，请先续费后再预约课程", gin.H{
			"membership_expire_at": member.MembershipExpireAt,
		})
	}

	var schedule models.Schedule
	if err := models.DB.Preload("Course").Preload("Coach").First(&schedule, scheduleID).Error; err != nil {
		return nil, NewServiceError(404, "课程排期不存在")
	}

	if schedule.StartTime.Before(time.Now()) {
		return nil, NewServiceError(400, "该课程已开始，无法预约")
	}

	var existingBooking models.Booking
	if err := models.DB.Where("schedule_id = ? AND member_id = ? AND status != ?",
		scheduleID, memberID, config.BookingStatusCanceled).First(&existingBooking).Error; err == nil {
		return nil, NewServiceError(400, "您已预约该课程，无需重复预约")
	}

	var existingWaitlist models.Waitlist
	if err := models.DB.Where("schedule_id = ? AND member_id = ? AND status = ?",
		scheduleID, memberID, models.WaitlistStatusWaiting).First(&existingWaitlist).Error; err == nil {
		return nil, NewServiceErrorWithData(400, "您已在候补队列中", gin.H{
			"position":   existingWaitlist.Position,
			"queue_type": "waitlist",
		})
	}

	bookedCount := schedule.BookedCount()
	if bookedCount >= schedule.Capacity {
		waitlistCount := schedule.WaitlistCount()
		position := waitlistCount + 1

		waitlist := models.Waitlist{
			ScheduleID: scheduleID,
			MemberID:   memberID,
			Position:   position,
			Status:     models.WaitlistStatusWaiting,
		}
		if err := models.DB.Create(&waitlist).Error; err != nil {
			return nil, NewServiceError(500, "候补失败: "+err.Error())
		}

		return &BookingResult{
			IsWaitlist:   true,
			Waitlist:     &waitlist,
			Schedule:     &schedule,
			Member:       &member,
			Position:     position,
			TotalWaiting: waitlistCount,
		}, nil
	}

	booking := models.Booking{
		ScheduleID: scheduleID,
		MemberID:   memberID,
		Status:     config.BookingStatusPending,
	}
	if err := models.DB.Create(&booking).Error; err != nil {
		return nil, NewServiceError(500, "预约失败: "+err.Error())
	}

	booking.Schedule = schedule
	booking.Member = member

	return &BookingResult{
		IsWaitlist: false,
		Booking:    &booking,
		Schedule:   &schedule,
		Member:     &member,
	}, nil
}

type CancelResult struct {
	Booking           *models.Booking
	PromotedBooking   *models.Booking
	PromotedWaitlist  *models.Waitlist
	PromotedMember    *models.User
	RemainingWaitlist int
}

func CancelBooking(bookingID, memberID uint, role string) (*CancelResult, *ServiceError) {
	var booking models.Booking
	query := models.DB.Preload("Schedule").Preload("Schedule.Course").Where("id = ?", bookingID)
	if role != config.RoleAdmin {
		query = query.Where("member_id = ?", memberID)
	}

	if err := query.First(&booking).Error; err != nil {
		return nil, NewServiceError(404, "预约不存在或无权限取消")
	}

	if booking.Status == config.BookingStatusCanceled {
		return nil, NewServiceError(400, "该预约已取消，无需重复操作")
	}

	if booking.Schedule.StartTime.Before(time.Now()) {
		return nil, NewServiceError(400, "课程已开始，无法取消预约")
	}

	booking.Status = config.BookingStatusCanceled
	models.DB.Save(&booking)

	result := &CancelResult{Booking: &booking}

	promotedBooking, promotedWaitlist, promoteErr := PromoteWaitlist(booking.ScheduleID)
	if promoteErr == nil && promotedBooking != nil {
		var promotedMember models.User
		models.DB.First(&promotedMember, promotedBooking.MemberID)

		result.PromotedBooking = promotedBooking
		result.PromotedWaitlist = promotedWaitlist
		result.PromotedMember = &promotedMember

		var remainingWaitlist []models.Waitlist
		models.DB.Where("schedule_id = ? AND status = ?", booking.ScheduleID, models.WaitlistStatusWaiting).
			Order("position ASC").Find(&remainingWaitlist)
		result.RemainingWaitlist = len(remainingWaitlist)
	}

	return result, nil
}

func PromoteWaitlist(scheduleID uint) (*models.Booking, *models.Waitlist, error) {
	var waitlist models.Waitlist
	if err := models.DB.Where("schedule_id = ? AND status = ?", scheduleID, models.WaitlistStatusWaiting).
		Order("position ASC").First(&waitlist).Error; err != nil {
		return nil, nil, err
	}

	waitlist.Status = models.WaitlistStatusPromoted
	models.DB.Save(&waitlist)

	booking := models.Booking{
		ScheduleID: scheduleID,
		MemberID:   waitlist.MemberID,
		Status:     config.BookingStatusPending,
	}
	if err := models.DB.Create(&booking).Error; err != nil {
		return nil, nil, err
	}

	ReorderWaitlistPositions(scheduleID, waitlist.ID)

	models.DB.Preload("Member").First(&waitlist, waitlist.ID)
	return &booking, &waitlist, nil
}

func ReorderWaitlistPositions(scheduleID uint, excludeID uint) {
	var remaining []models.Waitlist
	query := models.DB.Where("schedule_id = ? AND status = ?", scheduleID, models.WaitlistStatusWaiting)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	query.Order("position ASC").Find(&remaining)
	for i, w := range remaining {
		w.Position = i + 1
		models.DB.Save(&w)
	}
}

type CheckInResult struct {
	Booking *models.Booking
}

func CheckIn(bookingID uint) (*CheckInResult, *ServiceError) {
	var booking models.Booking
	if err := models.DB.Preload("Schedule").Preload("Schedule.Course").Preload("Member").First(&booking, bookingID).Error; err != nil {
		return nil, NewServiceError(404, "预约不存在")
	}

	if booking.Status == config.BookingStatusCanceled {
		return nil, NewServiceError(400, "该预约已取消，无法签到")
	}

	if booking.Status == config.BookingStatusChecked {
		return nil, NewServiceError(400, "该预约已签到，无需重复签到")
	}

	now := time.Now()
	startTime := booking.Schedule.StartTime
	endTime := booking.Schedule.EndTime

	if now.Before(startTime.Add(-30 * time.Minute)) {
		return nil, NewServiceErrorWithData(400, "签到时间未到，请在课程开始前30分钟内签到", gin.H{
			"start_time":   startTime,
			"current_time": now,
		})
	}

	if now.After(endTime) {
		return nil, NewServiceErrorWithData(400, "课程已结束，无法签到", gin.H{
			"end_time":     endTime,
			"current_time": now,
		})
	}

	nowPtr := now
	booking.Status = config.BookingStatusChecked
	booking.CheckedAt = &nowPtr
	models.DB.Save(&booking)

	return &CheckInResult{Booking: &booking}, nil
}

func GetMyBookings(memberID uint, status string) []gin.H {
	query := models.DB.Preload("Schedule").Preload("Schedule.Course").Preload("Schedule.Coach").
		Where("member_id = ?", memberID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var bookings []models.Booking
	query.Order("created_at DESC").Find(&bookings)

	var result []gin.H
	for _, b := range bookings {
		result = append(result, gin.H{
			"id":           b.ID,
			"schedule_id":  b.ScheduleID,
			"course_name":  b.Schedule.Course.Name,
			"coach_name":   b.Schedule.Coach.Name,
			"start_time":   b.Schedule.StartTime,
			"end_time":     b.Schedule.EndTime,
			"room":         b.Schedule.Room,
			"status":       b.Status,
			"checked_at":   b.CheckedAt,
			"created_at":   b.CreatedAt,
		})
	}
	return result
}

type BookingListResult struct {
	Total    int64
	Page     int
	PageSize int
	Bookings []gin.H
}

func GetBookingList(scheduleID, memberID, status string, page, pageSize int) *BookingListResult {
	offset := (page - 1) * pageSize

	query := models.DB.Preload("Schedule").Preload("Schedule.Course").Preload("Member").Model(&models.Booking{})

	if scheduleID != "" {
		query = query.Where("schedule_id = ?", scheduleID)
	}
	if memberID != "" {
		query = query.Where("member_id = ?", memberID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var bookings []models.Booking
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&bookings)

	var result []gin.H
	for _, b := range bookings {
		result = append(result, gin.H{
			"id":           b.ID,
			"schedule_id":  b.ScheduleID,
			"course_name":  b.Schedule.Course.Name,
			"member_id":    b.MemberID,
			"member_name":  b.Member.Name,
			"start_time":   b.Schedule.StartTime,
			"status":       b.Status,
			"checked_at":   b.CheckedAt,
			"created_at":   b.CreatedAt,
		})
	}

	return &BookingListResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Bookings: result,
	}
}

type ScheduleBookingsResult struct {
	Schedule gin.H
	Members  []gin.H
	Waitlist []gin.H
}

func GetScheduleBookings(scheduleID uint) (*ScheduleBookingsResult, *ServiceError) {
	var schedule models.Schedule
	if err := models.DB.Preload("Course").Preload("Coach").First(&schedule, scheduleID).Error; err != nil {
		return nil, NewServiceError(404, "排课不存在")
	}

	var bookings []models.Booking
	models.DB.Preload("Member").Where("schedule_id = ? AND status != ?", scheduleID, config.BookingStatusCanceled).
		Order("created_at ASC").Find(&bookings)

	var waitlist []models.Waitlist
	models.DB.Preload("Member").Where("schedule_id = ? AND status = ?", scheduleID, models.WaitlistStatusWaiting).
		Order("position ASC").Find(&waitlist)

	var memberList []gin.H
	checkedCount := 0
	for _, b := range bookings {
		if b.Status == config.BookingStatusChecked {
			checkedCount++
		}
		memberList = append(memberList, gin.H{
			"booking_id":  b.ID,
			"member_id":   b.MemberID,
			"member_name": b.Member.Name,
			"phone":       b.Member.Phone,
			"status":      b.Status,
			"checked_at":  b.CheckedAt,
			"booked_at":   b.CreatedAt,
		})
	}

	var waitlistList []gin.H
	for _, w := range waitlist {
		waitlistList = append(waitlistList, gin.H{
			"waitlist_id": w.ID,
			"member_id":   w.MemberID,
			"member_name": w.Member.Name,
			"phone":       w.Member.Phone,
			"position":    w.Position,
			"status":      w.Status,
			"created_at":  w.CreatedAt,
		})
	}

	return &ScheduleBookingsResult{
		Schedule: gin.H{
			"id":             schedule.ID,
			"course_name":    schedule.Course.Name,
			"coach_name":     schedule.Coach.Name,
			"start_time":     schedule.StartTime,
			"end_time":       schedule.EndTime,
			"capacity":       schedule.Capacity,
			"booked_count":   len(bookings),
			"waitlist_count": len(waitlist),
			"checked_count":  checkedCount,
			"room":           schedule.Room,
		},
		Members:  memberList,
		Waitlist: waitlistList,
	}, nil
}

func GetMyWaitlist(memberID uint) []gin.H {
	var waitlist []models.Waitlist
	models.DB.Preload("Schedule").Preload("Schedule.Course").Preload("Schedule.Coach").
		Where("member_id = ?", memberID).
		Order("created_at DESC").Find(&waitlist)

	var result []gin.H
	for _, w := range waitlist {
		result = append(result, gin.H{
			"id":          w.ID,
			"schedule_id": w.ScheduleID,
			"course_name": w.Schedule.Course.Name,
			"coach_name":  w.Schedule.Coach.Name,
			"start_time":  w.Schedule.StartTime,
			"position":    w.Position,
			"status":      w.Status,
			"created_at":  w.CreatedAt,
		})
	}
	return result
}

type CancelWaitlistResult struct {
	Waitlist          *models.Waitlist
	RemainingWaitlist int
}

func CancelWaitlist(waitlistID, memberID uint, role string) (*CancelWaitlistResult, *ServiceError) {
	var waitlist models.Waitlist
	query := models.DB.Preload("Schedule").Where("id = ?", waitlistID)
	if role != config.RoleAdmin {
		query = query.Where("member_id = ?", memberID)
	}

	if err := query.First(&waitlist).Error; err != nil {
		return nil, NewServiceError(404, "候补记录不存在或无权限取消")
	}

	if waitlist.Status != models.WaitlistStatusWaiting {
		return nil, NewServiceErrorWithData(400, "该候补无法取消", gin.H{
			"current_status": waitlist.Status,
		})
	}

	if waitlist.Schedule.StartTime.Before(time.Now()) {
		return nil, NewServiceError(400, "课程已开始，无法取消候补")
	}

	waitlist.Status = models.WaitlistStatusCanceled
	models.DB.Save(&waitlist)

	ReorderWaitlistPositions(waitlist.ScheduleID, 0)

	var remaining []models.Waitlist
	models.DB.Where("schedule_id = ? AND status = ?", waitlist.ScheduleID, models.WaitlistStatusWaiting).
		Order("position ASC").Find(&remaining)

	return &CancelWaitlistResult{
		Waitlist:          &waitlist,
		RemainingWaitlist: len(remaining),
	}, nil
}

func GetScheduleWaitlist(scheduleID uint) []gin.H {
	var waitlist []models.Waitlist
	models.DB.Preload("Member").Where("schedule_id = ? AND status = ?", scheduleID, models.WaitlistStatusWaiting).
		Order("position ASC").Find(&waitlist)

	var result []gin.H
	for _, w := range waitlist {
		result = append(result, gin.H{
			"waitlist_id": w.ID,
			"member_id":   w.MemberID,
			"member_name": w.Member.Name,
			"phone":       w.Member.Phone,
			"position":    w.Position,
			"status":      w.Status,
			"created_at":  w.CreatedAt,
		})
	}
	return result
}

func ParseUintParam(s string) (uint, error) {
	id, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
