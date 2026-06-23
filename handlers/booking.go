package handlers

import (
	"fitness-studio-api/config"
	"fitness-studio-api/middleware"
	"fitness-studio-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type BookingRequest struct {
	ScheduleID uint `json:"schedule_id" binding:"required"`
}

func CreateBooking(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)

	var req BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var member models.User
	if err := models.DB.Where("id = ? AND role = ?", memberID, config.RoleMember).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会员不存在"})
		return
	}

	if member.MembershipStatus() == config.MembershipStatusExpired {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "会员已过期，请先续费后再预约课程",
			"membership_expire_at": member.MembershipExpireAt,
		})
		return
	}

	var schedule models.Schedule
	if err := models.DB.Preload("Course").Preload("Coach").First(&schedule, req.ScheduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "课程排期不存在"})
		return
	}

	if schedule.StartTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该课程已开始，无法预约"})
		return
	}

	bookedCount := schedule.BookedCount()
	if bookedCount >= schedule.Capacity {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "该课程已满员，无法预约",
			"capacity":      schedule.Capacity,
			"booked_count":  bookedCount,
			"course_name":   schedule.Course.Name,
			"start_time":    schedule.StartTime,
		})
		return
	}

	var existingBooking models.Booking
	if err := models.DB.Where("schedule_id = ? AND member_id = ? AND status != ?",
		req.ScheduleID, memberID, config.BookingStatusCanceled).First(&existingBooking).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "您已预约该课程，无需重复预约"})
		return
	}

	booking := models.Booking{
		ScheduleID: req.ScheduleID,
		MemberID:   memberID,
		Status:     config.BookingStatusPending,
	}

	if err := models.DB.Create(&booking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "预约失败: " + err.Error()})
		return
	}

	booking.Schedule = schedule
	booking.Member = member

	c.JSON(http.StatusCreated, gin.H{
		"message": "预约成功",
		"booking": gin.H{
			"id":           booking.ID,
			"schedule_id":  booking.ScheduleID,
			"course_name":  schedule.Course.Name,
			"coach_name":   schedule.Coach.Name,
			"start_time":   schedule.StartTime,
			"end_time":     schedule.EndTime,
			"room":         schedule.Room,
			"status":       booking.Status,
			"created_at":   booking.CreatedAt,
		},
	})
}

func CancelBooking(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentUserRole(c)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}

	var booking models.Booking
	query := models.DB.Preload("Schedule").Preload("Schedule.Course").Where("id = ?", uint(id))
	if role != config.RoleAdmin {
		query = query.Where("member_id = ?", memberID)
	}

	if err := query.First(&booking).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在或无权限取消"})
		return
	}

	if booking.Status == config.BookingStatusCanceled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该预约已取消，无需重复操作"})
		return
	}

	if booking.Schedule.StartTime.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "课程已开始，无法取消预约"})
		return
	}

	booking.Status = config.BookingStatusCanceled
	models.DB.Save(&booking)

	c.JSON(http.StatusOK, gin.H{
		"message": "取消预约成功",
		"booking": gin.H{
			"id":          booking.ID,
			"course_name": booking.Schedule.Course.Name,
			"start_time":  booking.Schedule.StartTime,
			"status":      booking.Status,
		},
	})
}

func CheckIn(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}

	var booking models.Booking
	if err := models.DB.Preload("Schedule").Preload("Schedule.Course").Preload("Member").First(&booking, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "预约不存在"})
		return
	}

	if booking.Status == config.BookingStatusCanceled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该预约已取消，无法签到"})
		return
	}

	if booking.Status == config.BookingStatusChecked {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该预约已签到，无需重复签到"})
		return
	}

	now := time.Now()
	startTime := booking.Schedule.StartTime
	endTime := booking.Schedule.EndTime

	if now.Before(startTime.Add(-30 * time.Minute)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "签到时间未到，请在课程开始前30分钟内签到",
			"start_time": startTime,
			"current_time": now,
		})
		return
	}

	if now.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "课程已结束，无法签到",
			"end_time": endTime,
			"current_time": now,
		})
		return
	}

	nowPtr := now
	booking.Status = config.BookingStatusChecked
	booking.CheckedAt = &nowPtr
	models.DB.Save(&booking)

	c.JSON(http.StatusOK, gin.H{
		"message": "签到成功",
		"booking": gin.H{
			"id":           booking.ID,
			"member_name":  booking.Member.Name,
			"course_name":  booking.Schedule.Course.Name,
			"start_time":   booking.Schedule.StartTime,
			"checked_at":   booking.CheckedAt,
			"status":       booking.Status,
		},
	})
}

func GetMyBookings(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)
	status := c.Query("status")

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

	c.JSON(http.StatusOK, gin.H{
		"count":   len(result),
		"bookings": result,
	})
}

func GetBookingList(c *gin.Context) {
	scheduleID := c.Query("schedule_id")
	memberID := c.Query("member_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

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

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"bookings":  result,
	})
}

func GetScheduleBookings(c *gin.Context) {
	scheduleIDParam := c.Param("id")
	scheduleID, err := strconv.ParseUint(scheduleIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	var schedule models.Schedule
	if err := models.DB.Preload("Course").Preload("Coach").First(&schedule, uint(scheduleID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排课不存在"})
		return
	}

	var bookings []models.Booking
	models.DB.Preload("Member").Where("schedule_id = ? AND status != ?", uint(scheduleID), config.BookingStatusCanceled).
		Order("created_at ASC").Find(&bookings)

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

	c.JSON(http.StatusOK, gin.H{
		"schedule": gin.H{
			"id":            schedule.ID,
			"course_name":   schedule.Course.Name,
			"coach_name":    schedule.Coach.Name,
			"start_time":    schedule.StartTime,
			"end_time":      schedule.EndTime,
			"capacity":      schedule.Capacity,
			"booked_count":  len(bookings),
			"checked_count": checkedCount,
			"room":          schedule.Room,
		},
		"members": memberList,
	})
}
