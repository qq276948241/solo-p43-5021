package handlers

import (
	"fitness-studio-api/config"
	"fitness-studio-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetWeeklyAttendance(c *gin.Context) {
	dateStr := c.Query("date")

	var refDate time.Time
	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			refDate = time.Now()
		} else {
			refDate = parsed
		}
	} else {
		refDate = time.Now()
	}

	weekStart := config.GetStartOfWeek(refDate)
	weekEnd := config.GetEndOfWeek(refDate)

	type ScheduleStats struct {
		CourseID   uint
		CourseName string
		Capacity   int
	}

	var scheduleStats []ScheduleStats
	models.DB.Table("schedules").
		Select("schedules.course_id, courses.name as course_name, schedules.capacity").
		Joins("left join courses on schedules.course_id = courses.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", weekStart, weekEnd).
		Scan(&scheduleStats)

	courseStatsMap := make(map[uint]models.WeeklyAttendance)
	totalSlots := 0
	totalBooked := 0
	totalChecked := 0

	for _, ss := range scheduleStats {
		stat, exists := courseStatsMap[ss.CourseID]
		if !exists {
			stat = models.WeeklyAttendance{
				WeekStart:  weekStart,
				CourseName: ss.CourseName,
			}
		}
		stat.TotalSlots += ss.Capacity
		totalSlots += ss.Capacity
		courseStatsMap[ss.CourseID] = stat
	}

	type BookingCount struct {
		CourseID    uint
		BookedCount int
	}

	var bookedCounts []BookingCount
	models.DB.Table("bookings").
		Select("schedules.course_id, count(bookings.id) as booked_count").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", weekStart, weekEnd).
		Where("bookings.status != ?", config.BookingStatusCanceled).
		Group("schedules.course_id").
		Scan(&bookedCounts)

	for _, bc := range bookedCounts {
		if stat, exists := courseStatsMap[bc.CourseID]; exists {
			stat.BookedCount = bc.BookedCount
			totalBooked += bc.BookedCount
			courseStatsMap[bc.CourseID] = stat
		}
	}

	var checkedCounts []BookingCount
	models.DB.Table("bookings").
		Select("schedules.course_id, count(bookings.id) as booked_count").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", weekStart, weekEnd).
		Where("bookings.status = ?", config.BookingStatusChecked).
		Group("schedules.course_id").
		Scan(&checkedCounts)

	for _, cc := range checkedCounts {
		if stat, exists := courseStatsMap[cc.CourseID]; exists {
			stat.CheckedCount = cc.BookedCount
			totalChecked += cc.BookedCount
			if stat.TotalSlots > 0 {
				stat.AttendanceRate = float64(stat.CheckedCount) / float64(stat.TotalSlots) * 100
			}
			courseStatsMap[cc.CourseID] = stat
		}
	}

	var courseStats []models.WeeklyAttendance
	for _, stat := range courseStatsMap {
		if stat.TotalSlots > 0 && stat.AttendanceRate == 0 {
			stat.AttendanceRate = float64(stat.CheckedCount) / float64(stat.TotalSlots) * 100
		}
		courseStats = append(courseStats, stat)
	}

	var overallRate float64
	if totalSlots > 0 {
		overallRate = float64(totalChecked) / float64(totalSlots) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"week_start": weekStart,
		"week_end":   weekEnd,
		"summary": gin.H{
			"total_slots":       totalSlots,
			"total_booked":      totalBooked,
			"total_checked":     totalChecked,
			"booking_rate":      float64(totalBooked) / float64(totalSlots) * 100,
			"attendance_rate":   overallRate,
		},
		"course_attendance": courseStats,
	})
}

func GetMemberActivity(c *gin.Context) {
	dateStr := c.Query("date")

	var refDate time.Time
	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			refDate = time.Now()
		} else {
			refDate = parsed
		}
	} else {
		refDate = time.Now()
	}

	weekStart := config.GetStartOfWeek(refDate)
	weekEnd := config.GetEndOfWeek(refDate)

	type MemberActivityResult struct {
		MemberID   uint
		MemberName string
		BookCount  int
		CheckCount int
	}

	var activityResults []MemberActivityResult

	models.DB.Table("bookings").
		Select("bookings.member_id, users.name as member_name, count(bookings.id) as book_count, sum(case when bookings.status = 'checked' then 1 else 0 end) as check_count").
		Joins("left join users on bookings.member_id = users.id").
		Where("bookings.created_at >= ? AND bookings.created_at <= ?", weekStart, weekEnd).
		Where("bookings.status != ?", config.BookingStatusCanceled).
		Group("bookings.member_id, users.name").
		Order("check_count DESC, book_count DESC").
		Limit(20).
		Scan(&activityResults)

	var memberActivities []models.MemberActivity
	for _, ar := range activityResults {
		activityScore := ar.CheckCount*10 + ar.BookCount*3
		memberActivities = append(memberActivities, models.MemberActivity{
			MemberID:      ar.MemberID,
			MemberName:    ar.MemberName,
			BookCount:     ar.BookCount,
			CheckCount:    ar.CheckCount,
			ActivityScore: activityScore,
		})
	}

	var totalMembers int64
	models.DB.Model(&models.User{}).Where("role = ? AND membership_expire_at > ?", config.RoleMember, time.Now()).Count(&totalMembers)

	var totalBookings int64
	models.DB.Model(&models.Booking{}).
		Where("created_at >= ? AND created_at <= ?", weekStart, weekEnd).
		Where("status != ?", config.BookingStatusCanceled).
		Count(&totalBookings)

	var totalCheckins int64
	models.DB.Model(&models.Booking{}).
		Where("created_at >= ? AND created_at <= ?", weekStart, weekEnd).
		Where("status = ?", config.BookingStatusChecked).
		Count(&totalCheckins)

	c.JSON(http.StatusOK, gin.H{
		"week_start": weekStart,
		"week_end":   weekEnd,
		"summary": gin.H{
			"active_members":   totalMembers,
			"total_bookings":   totalBookings,
			"total_checkins":   totalCheckins,
			"avg_bookings_per_member": float64(totalBookings) / float64(totalMembers),
		},
		"member_activities": memberActivities,
	})
}

func GetDashboardStats(c *gin.Context) {
	now := time.Now()
	weekStart := config.GetStartOfWeek(now)
	weekEnd := config.GetEndOfWeek(now)

	var totalMembers int64
	models.DB.Model(&models.User{}).Where("role = ?", config.RoleMember).Count(&totalMembers)

	var activeMembers int64
	models.DB.Model(&models.User{}).Where("role = ? AND membership_expire_at > ?", config.RoleMember, now).Count(&activeMembers)

	var expiringSoon int64
	models.DB.Model(&models.User{}).
		Where("role = ? AND membership_expire_at >= ? AND membership_expire_at <= ?",
			config.RoleMember, now, now.AddDate(0, 0, 7)).
		Count(&expiringSoon)

	var todaySchedules int64
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
	models.DB.Model(&models.Schedule{}).
		Where("start_time >= ? AND start_time <= ?", todayStart, todayEnd).
		Count(&todaySchedules)

	var todayBookings int64
	models.DB.Table("bookings").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", todayStart, todayEnd).
		Where("bookings.status != ?", config.BookingStatusCanceled).
		Count(&todayBookings)

	var todayCheckins int64
	models.DB.Table("bookings").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", todayStart, todayEnd).
		Where("bookings.status = ?", config.BookingStatusChecked).
		Count(&todayCheckins)

	var weekBookings int64
	models.DB.Table("bookings").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", weekStart, weekEnd).
		Where("bookings.status != ?", config.BookingStatusCanceled).
		Count(&weekBookings)

	var weekCheckins int64
	models.DB.Table("bookings").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Where("schedules.start_time >= ? AND schedules.start_time <= ?", weekStart, weekEnd).
		Where("bookings.status = ?", config.BookingStatusChecked).
		Count(&weekCheckins)

	type TopCourse struct {
		CourseName string
		BookCount  int
	}
	var topCourses []TopCourse
	models.DB.Table("bookings").
		Select("courses.name as course_name, count(bookings.id) as book_count").
		Joins("left join schedules on bookings.schedule_id = schedules.id").
		Joins("left join courses on schedules.course_id = courses.id").
		Where("bookings.status != ?", config.BookingStatusCanceled).
		Group("courses.name").
		Order("book_count DESC").
		Limit(5).
		Scan(&topCourses)

	c.JSON(http.StatusOK, gin.H{
		"members": gin.H{
			"total":          totalMembers,
			"active":         activeMembers,
			"expiring_soon":  expiringSoon,
		},
		"today": gin.H{
			"schedules":     todaySchedules,
			"bookings":      todayBookings,
			"checkins":      todayCheckins,
		},
		"this_week": gin.H{
			"bookings":      weekBookings,
			"checkins":      weekCheckins,
		},
		"top_courses": topCourses,
	})
}
