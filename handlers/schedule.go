package handlers

import (
	"fitness-studio-api/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CoachRequest struct {
	Name      string `json:"name" binding:"required"`
	Phone     string `json:"phone"`
	Specialty string `json:"specialty"`
}

func GetCoachList(c *gin.Context) {
	var coaches []models.Coach
	models.DB.Find(&coaches)

	c.JSON(http.StatusOK, gin.H{
		"coaches": coaches,
	})
}

func CreateCoach(c *gin.Context) {
	var req CoachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	coach := models.Coach{
		Name:      req.Name,
		Phone:     req.Phone,
		Specialty: req.Specialty,
	}

	if err := models.DB.Create(&coach).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建教练失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"coach":   coach,
	})
}

func UpdateCoach(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的教练ID"})
		return
	}

	var coach models.Coach
	if err := models.DB.First(&coach, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "教练不存在"})
		return
	}

	var req CoachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	coach.Name = req.Name
	coach.Phone = req.Phone
	coach.Specialty = req.Specialty
	models.DB.Save(&coach)

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"coach":   coach,
	})
}

func DeleteCoach(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的教练ID"})
		return
	}

	var coach models.Coach
	if err := models.DB.First(&coach, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "教练不存在"})
		return
	}

	models.DB.Delete(&coach)

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

type CourseRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Duration    int    `json:"duration" binding:"required,min=1"`
}

func GetCourseList(c *gin.Context) {
	var courses []models.Course
	models.DB.Find(&courses)

	c.JSON(http.StatusOK, gin.H{
		"courses": courses,
	})
}

func CreateCourse(c *gin.Context) {
	var req CourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	course := models.Course{
		Name:        req.Name,
		Description: req.Description,
		Duration:    req.Duration,
	}

	if err := models.DB.Create(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建课程失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"course":  course,
	})
}

func UpdateCourse(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的课程ID"})
		return
	}

	var course models.Course
	if err := models.DB.First(&course, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "课程不存在"})
		return
	}

	var req CourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	course.Name = req.Name
	course.Description = req.Description
	course.Duration = req.Duration
	models.DB.Save(&course)

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"course":  course,
	})
}

func DeleteCourse(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的课程ID"})
		return
	}

	var course models.Course
	if err := models.DB.First(&course, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "课程不存在"})
		return
	}

	models.DB.Delete(&course)

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

type ScheduleRequest struct {
	CourseID  uint      `json:"course_id" binding:"required"`
	CoachID   uint      `json:"coach_id" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`
	Capacity  int       `json:"capacity" binding:"required,min=1"`
	Room      string    `json:"room"`
}

func GetScheduleList(c *gin.Context) {
	dateStr := c.Query("date")
	courseID := c.Query("course_id")
	coachID := c.Query("coach_id")

	query := models.DB.Preload("Course").Preload("Coach").Model(&models.Schedule{})

	if dateStr != "" {
		date, err := time.Parse("2006-01-02", dateStr)
		if err == nil {
			startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
			endOfDay := startOfDay.AddDate(0, 0, 1).Add(-time.Nanosecond)
			query = query.Where("start_time >= ? AND start_time <= ?", startOfDay, endOfDay)
		}
	}

	if courseID != "" {
		query = query.Where("course_id = ?", courseID)
	}

	if coachID != "" {
		query = query.Where("coach_id = ?", coachID)
	}

	var schedules []models.Schedule
	query.Order("start_time ASC").Find(&schedules)

	var result []gin.H
	for _, s := range schedules {
		bookedCount := s.BookedCount()
		result = append(result, gin.H{
			"id":             s.ID,
			"course_id":      s.CourseID,
			"course_name":    s.Course.Name,
			"coach_id":       s.CoachID,
			"coach_name":     s.Coach.Name,
			"start_time":     s.StartTime,
			"end_time":       s.EndTime,
			"capacity":       s.Capacity,
			"booked_count":   bookedCount,
			"available_spots": s.Capacity - bookedCount,
			"room":           s.Room,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": result,
	})
}

func GetScheduleDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	var schedule models.Schedule
	if err := models.DB.Preload("Course").Preload("Coach").First(&schedule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排课不存在"})
		return
	}

	bookedCount := schedule.BookedCount()

	c.JSON(http.StatusOK, gin.H{
		"id":              schedule.ID,
		"course_id":       schedule.CourseID,
		"course_name":     schedule.Course.Name,
		"course_desc":     schedule.Course.Description,
		"coach_id":        schedule.CoachID,
		"coach_name":      schedule.Coach.Name,
		"coach_specialty": schedule.Coach.Specialty,
		"start_time":      schedule.StartTime,
		"end_time":        schedule.EndTime,
		"capacity":        schedule.Capacity,
		"booked_count":    bookedCount,
		"available_spots": schedule.Capacity - bookedCount,
		"room":            schedule.Room,
	})
}

func CreateSchedule(c *gin.Context) {
	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	var course models.Course
	if err := models.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "课程不存在"})
		return
	}

	var coach models.Coach
	if err := models.DB.First(&coach, req.CoachID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "教练不存在"})
		return
	}

	if !req.EndTime.After(req.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "结束时间必须晚于开始时间"})
		return
	}

	schedule := models.Schedule{
		CourseID:  req.CourseID,
		CoachID:   req.CoachID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Capacity:  req.Capacity,
		Room:      req.Room,
	}

	if err := models.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建排课失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"schedule": gin.H{
			"id":         schedule.ID,
			"course_id":  schedule.CourseID,
			"coach_id":   schedule.CoachID,
			"start_time": schedule.StartTime,
			"end_time":   schedule.EndTime,
			"capacity":   schedule.Capacity,
			"room":       schedule.Room,
		},
	})
}

func UpdateSchedule(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	var schedule models.Schedule
	if err := models.DB.First(&schedule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排课不存在"})
		return
	}

	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	schedule.CourseID = req.CourseID
	schedule.CoachID = req.CoachID
	schedule.StartTime = req.StartTime
	schedule.EndTime = req.EndTime
	schedule.Capacity = req.Capacity
	schedule.Room = req.Room

	models.DB.Save(&schedule)

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

func DeleteSchedule(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的排课ID"})
		return
	}

	var schedule models.Schedule
	if err := models.DB.First(&schedule, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "排课不存在"})
		return
	}

	models.DB.Delete(&schedule)

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}
