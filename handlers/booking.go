package handlers

import (
	"fitness-studio-api/middleware"
	"fitness-studio-api/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type BookingRequest struct {
	ScheduleID uint `json:"schedule_id" binding:"required"`
}

func handleServiceError(c *gin.Context, svcErr *service.ServiceError) {
	resp := gin.H{"error": svcErr.Message}
	for k, v := range svcErr.Data {
		resp[k] = v
	}
	c.JSON(svcErr.Status, resp)
}

func CreateBooking(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)

	var req BookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	result, svcErr := service.CreateBooking(memberID, req.ScheduleID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	if result.IsWaitlist {
		c.JSON(http.StatusOK, gin.H{
			"message":    "课程已满，已加入候补队列",
			"queue_type": "waitlist",
			"waitlist": gin.H{
				"id":            result.Waitlist.ID,
				"schedule_id":   result.Waitlist.ScheduleID,
				"course_name":   result.Schedule.Course.Name,
				"coach_name":    result.Schedule.Coach.Name,
				"start_time":    result.Schedule.StartTime,
				"position":      result.Position,
				"total_waiting": result.TotalWaiting,
				"status":        result.Waitlist.Status,
				"created_at":    result.Waitlist.CreatedAt,
			},
			"note": "有人取消预约时，将按候补顺序自动递补",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "预约成功",
		"booking": gin.H{
			"id":           result.Booking.ID,
			"schedule_id":  result.Booking.ScheduleID,
			"course_name":  result.Schedule.Course.Name,
			"coach_name":   result.Schedule.Coach.Name,
			"start_time":   result.Schedule.StartTime,
			"end_time":     result.Schedule.EndTime,
			"room":         result.Schedule.Room,
			"status":       result.Booking.Status,
			"created_at":   result.Booking.CreatedAt,
		},
	})
}

func CancelBooking(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentUserRole(c)

	id, err := service.ParseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}

	result, svcErr := service.CancelBooking(id, memberID, role)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response := gin.H{
		"message": "取消预约成功",
		"booking": gin.H{
			"id":          result.Booking.ID,
			"course_name": result.Booking.Schedule.Course.Name,
			"start_time":  result.Booking.Schedule.StartTime,
			"status":      result.Booking.Status,
		},
	}

	if result.PromotedBooking != nil {
		response["promoted"] = gin.H{
			"message":        "已自动递补候补会员",
			"member_id":      result.PromotedBooking.MemberID,
			"member_name":    result.PromotedMember.Name,
			"new_booking_id": result.PromotedBooking.ID,
			"waitlist_id":    result.PromotedWaitlist.ID,
			"was_position":   result.PromotedWaitlist.Position,
		}
		response["remaining_waitlist"] = result.RemainingWaitlist
	}

	c.JSON(http.StatusOK, response)
}

func CheckIn(c *gin.Context) {
	id, err := service.ParseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的预约ID"})
		return
	}

	result, svcErr := service.CheckIn(id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	b := result.Booking
	c.JSON(http.StatusOK, gin.H{
		"message": "签到成功",
		"booking": gin.H{
			"id":           b.ID,
			"member_name":  b.Member.Name,
			"course_name":  b.Schedule.Course.Name,
			"start_time":   b.Schedule.StartTime,
			"checked_at":   b.CheckedAt,
			"status":       b.Status,
		},
	})
}

func GetMyBookings(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)
	status := c.Query("status")

	bookings := service.GetMyBookings(memberID, status)

	c.JSON(http.StatusOK, gin.H{
		"count":    len(bookings),
		"bookings": bookings,
	})
}

func GetBookingList(c *gin.Context) {
	scheduleID := c.Query("schedule_id")
	memberID := c.Query("member_id")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result := service.GetBookingList(scheduleID, memberID, status, page, pageSize)

	c.JSON(http.StatusOK, gin.H{
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
		"bookings":  result.Bookings,
	})
}

func GetScheduleBookings(c *gin.Context) {
	scheduleID, err := service.ParseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	result, svcErr := service.GetScheduleBookings(scheduleID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedule": result.Schedule,
		"members":  result.Members,
		"waitlist": result.Waitlist,
	})
}

func GetMyWaitlist(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)

	waitlist := service.GetMyWaitlist(memberID)

	c.JSON(http.StatusOK, gin.H{
		"count":    len(waitlist),
		"waitlist": waitlist,
	})
}

func CancelWaitlist(c *gin.Context) {
	memberID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentUserRole(c)

	id, err := service.ParseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的候补ID"})
		return
	}

	result, svcErr := service.CancelWaitlist(id, memberID, role)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "取消候补成功",
		"waitlist": gin.H{
			"id":          result.Waitlist.ID,
			"schedule_id": result.Waitlist.ScheduleID,
			"status":      result.Waitlist.Status,
		},
		"remaining_waitlist": result.RemainingWaitlist,
	})
}

func GetScheduleWaitlist(c *gin.Context) {
	scheduleID, err := service.ParseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	waitlist := service.GetScheduleWaitlist(scheduleID)

	c.JSON(http.StatusOK, gin.H{
		"count":    len(waitlist),
		"waitlist": waitlist,
	})
}
