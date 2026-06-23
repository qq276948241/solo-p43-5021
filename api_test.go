package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"fitness-studio-api/config"
	"fitness-studio-api/handlers"
	"fitness-studio-api/middleware"
	"fitness-studio-api/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var testRouter *gin.Engine
var adminToken string
var memberToken string
var testScheduleID uint
var testBookingID uint

func setupTestRouter() {
	gin.SetMode(gin.TestMode)

	config.AppConfig.DBPath = "./test_fitness.db"
	config.AppConfig.JWTSecret = "test-secret-key"
	config.AppConfig.JWTExpireHour = 24

	models.InitDB()

	testRouter = gin.Default()

	testRouter.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := testRouter.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/register", handlers.Register)
			auth.GET("/me", middleware.AuthMiddleware(), handlers.GetCurrentUser)
		}

		members := api.Group("/members")
		members.Use(middleware.AuthMiddleware())
		{
			members.GET("", middleware.AdminRequired(), handlers.GetMemberList)
			members.GET("/expiring-soon", middleware.AdminRequired(), handlers.GetMemberExpiringSoon)
			members.GET("/:id", middleware.AdminRequired(), handlers.GetMemberDetail)
			members.PUT("/:id", handlers.UpdateMember)
			members.DELETE("/:id", middleware.AdminRequired(), handlers.DeleteMember)
			members.POST("/renew", handlers.RenewMembership)
			members.POST("/:id/renew", middleware.AdminRequired(), handlers.RenewMembership)
		}

		coaches := api.Group("/coaches")
		coaches.Use(middleware.AuthMiddleware())
		{
			coaches.GET("", handlers.GetCoachList)
			coaches.POST("", middleware.AdminRequired(), handlers.CreateCoach)
			coaches.PUT("/:id", middleware.AdminRequired(), handlers.UpdateCoach)
			coaches.DELETE("/:id", middleware.AdminRequired(), handlers.DeleteCoach)
		}

		courses := api.Group("/courses")
		courses.Use(middleware.AuthMiddleware())
		{
			courses.GET("", handlers.GetCourseList)
			courses.POST("", middleware.AdminRequired(), handlers.CreateCourse)
			courses.PUT("/:id", middleware.AdminRequired(), handlers.UpdateCourse)
			courses.DELETE("/:id", middleware.AdminRequired(), handlers.DeleteCourse)
		}

		schedules := api.Group("/schedules")
		schedules.Use(middleware.AuthMiddleware())
		{
			schedules.GET("", handlers.GetScheduleList)
			schedules.GET("/:id", handlers.GetScheduleDetail)
			schedules.POST("", middleware.AdminRequired(), handlers.CreateSchedule)
			schedules.PUT("/:id", middleware.AdminRequired(), handlers.UpdateSchedule)
			schedules.DELETE("/:id", middleware.AdminRequired(), handlers.DeleteSchedule)
			schedules.GET("/:id/bookings", middleware.AdminRequired(), handlers.GetScheduleBookings)
		}

		bookings := api.Group("/bookings")
		bookings.Use(middleware.AuthMiddleware())
		{
			bookings.GET("", middleware.AdminRequired(), handlers.GetBookingList)
			bookings.GET("/my", handlers.GetMyBookings)
			bookings.POST("", handlers.CreateBooking)
			bookings.PUT("/:id/cancel", handlers.CancelBooking)
			bookings.PUT("/:id/checkin", middleware.AdminRequired(), handlers.CheckIn)
		}

		waitlist := api.Group("/waitlist")
		waitlist.Use(middleware.AuthMiddleware())
		{
			waitlist.GET("/my", handlers.GetMyWaitlist)
			waitlist.PUT("/:id/cancel", handlers.CancelWaitlist)
			waitlist.GET("/schedule/:id", middleware.AdminRequired(), handlers.GetScheduleWaitlist)
		}

		stats := api.Group("/stats")
		stats.Use(middleware.AuthMiddleware(), middleware.AdminRequired())
		{
			stats.GET("/dashboard", handlers.GetDashboardStats)
			stats.GET("/weekly-attendance", handlers.GetWeeklyAttendance)
			stats.GET("/member-activity", handlers.GetMemberActivity)
		}
	}

	testRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}

func TestMain(m *testing.M) {
	setupTestRouter()
	code := m.Run()
	os.Remove("./test_fitness.db")
	os.Exit(code)
}

func makeRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reader = bytes.NewReader(jsonBody)
	}

	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func parseResponse(w *httptest.ResponseRecorder, v interface{}) {
	json.Unmarshal(w.Body.Bytes(), v)
}

func Test01_HealthCheck(t *testing.T) {
	w := makeRequest("GET", "/health", nil, "")
	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, "ok", resp["status"])
	fmt.Println("✓ Health check passed")
}

func Test02_AdminLogin(t *testing.T) {
	body := map[string]string{"username": "admin", "password": "admin123"}
	w := makeRequest("POST", "/api/auth/login", body, "")
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["token"])
	adminToken = resp["token"].(string)
	assert.Equal(t, "admin", resp["user"].(map[string]interface{})["username"])
	fmt.Println("✓ Admin login passed")
}

func Test03_MemberLogin(t *testing.T) {
	body := map[string]string{"username": "member1", "password": "123456"}
	w := makeRequest("POST", "/api/auth/login", body, "")
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["token"])
	memberToken = resp["token"].(string)
	fmt.Println("✓ Member login passed")
}

func Test04_InvalidLogin(t *testing.T) {
	body := map[string]string{"username": "admin", "password": "wrong"}
	w := makeRequest("POST", "/api/auth/login", body, "")
	assert.Equal(t, 401, w.Code)
	fmt.Println("✓ Invalid login rejected")
}

func Test05_NoTokenRejected(t *testing.T) {
	w := makeRequest("GET", "/api/members", nil, "")
	assert.Equal(t, 401, w.Code)
	fmt.Println("✓ No token rejected")
}

func Test06_MemberForbiddenFromAdminEndpoint(t *testing.T) {
	w := makeRequest("GET", "/api/members", nil, memberToken)
	assert.Equal(t, 403, w.Code)
	fmt.Println("✓ Member forbidden from admin endpoint")
}

func Test07_GetCurrentUser(t *testing.T) {
	w := makeRequest("GET", "/api/auth/me", nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	user := resp["user"].(map[string]interface{})
	assert.Equal(t, "王小明", user["name"])
	assert.Equal(t, "member", user["role"])
	fmt.Println("✓ Get current user passed")
}

func Test08_GetMemberList(t *testing.T) {
	w := makeRequest("GET", "/api/members?page=1&page_size=10", nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.GreaterOrEqual(t, int(resp["total"].(float64)), 2)
	fmt.Println("✓ Get member list passed")
}

func Test09_GetCoaches(t *testing.T) {
	w := makeRequest("GET", "/api/coaches", nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	coaches := resp["coaches"].([]interface{})
	assert.GreaterOrEqual(t, len(coaches), 3)
	fmt.Println("✓ Get coaches passed")
}

func Test10_GetCourses(t *testing.T) {
	w := makeRequest("GET", "/api/courses", nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	courses := resp["courses"].([]interface{})
	assert.GreaterOrEqual(t, len(courses), 4)
	fmt.Println("✓ Get courses passed")
}

func Test11_GetSchedules(t *testing.T) {
	w := makeRequest("GET", "/api/schedules", nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	schedules := resp["schedules"].([]interface{})
	assert.GreaterOrEqual(t, len(schedules), 4)

	now := time.Now()
	for _, s := range schedules {
		sched := s.(map[string]interface{})
		startTimeStr := sched["start_time"].(string)
		startTime, _ := time.Parse(time.RFC3339, startTimeStr)
		if startTime.After(now) {
			testScheduleID = uint(sched["id"].(float64))
			break
		}
	}

	if testScheduleID == 0 {
		startTime := time.Now().AddDate(0, 0, 1)
		endTime := startTime.Add(time.Hour)
		body := map[string]interface{}{
			"course_id":  1,
			"coach_id":   1,
			"start_time": startTime,
			"end_time":   endTime,
			"capacity":   10,
			"room":       "测试预约房",
		}
		w2 := makeRequest("POST", "/api/schedules", body, adminToken)
		var resp2 map[string]interface{}
		parseResponse(w2, &resp2)
		if resp2["schedule"] != nil {
			testScheduleID = uint(resp2["schedule"].(map[string]interface{})["id"].(float64))
		}
	}

	assert.NotZero(t, testScheduleID)
	fmt.Println("✓ Get schedules passed, using schedule ID:", testScheduleID)
}

func Test12_CreateBooking(t *testing.T) {
	body := map[string]uint{"schedule_id": testScheduleID}
	w := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, "预约成功", resp["message"])
	booking := resp["booking"].(map[string]interface{})
	testBookingID = uint(booking["id"].(float64))
	fmt.Println("✓ Create booking passed, booking ID:", testBookingID)
}

func Test13_DuplicateBookingRejected(t *testing.T) {
	body := map[string]uint{"schedule_id": testScheduleID}
	w := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 400, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Contains(t, resp["error"], "重复")
	fmt.Println("✓ Duplicate booking rejected")
}

func Test14_GetMyBookings(t *testing.T) {
	w := makeRequest("GET", "/api/bookings/my", nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	bookings := resp["bookings"].([]interface{})
	assert.GreaterOrEqual(t, len(bookings), 1)
	fmt.Println("✓ Get my bookings passed")
}

func Test15_CancelBooking(t *testing.T) {
	w := makeRequest("PUT", fmt.Sprintf("/api/bookings/%d/cancel", testBookingID), nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	booking := resp["booking"].(map[string]interface{})
	assert.Equal(t, "canceled", booking["status"])
	fmt.Println("✓ Cancel booking passed")
}

func Test16_RebookAfterCancel(t *testing.T) {
	body := map[string]uint{"schedule_id": testScheduleID}
	w := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	booking := resp["booking"].(map[string]interface{})
	testBookingID = uint(booking["id"].(float64))
	fmt.Println("✓ Rebook after cancel passed, new booking ID:", testBookingID)
}

func Test17_CheckIn(t *testing.T) {
	startTime := time.Now().Add(10 * time.Minute)
	endTime := startTime.Add(time.Hour)
	scheduleBody := map[string]interface{}{
		"course_id":  1,
		"coach_id":   1,
		"start_time": startTime,
		"end_time":   endTime,
		"capacity":   10,
		"room":       "签到测试房",
	}
	w1 := makeRequest("POST", "/api/schedules", scheduleBody, adminToken)
	var resp1 map[string]interface{}
	parseResponse(w1, &resp1)
	checkinScheduleID := uint(resp1["schedule"].(map[string]interface{})["id"].(float64))

	bookingBody := map[string]uint{"schedule_id": checkinScheduleID}
	w2 := makeRequest("POST", "/api/bookings", bookingBody, memberToken)
	var resp2 map[string]interface{}
	parseResponse(w2, &resp2)
	checkinBookingID := uint(resp2["booking"].(map[string]interface{})["id"].(float64))

	w := makeRequest("PUT", fmt.Sprintf("/api/bookings/%d/checkin", checkinBookingID), nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	booking := resp["booking"].(map[string]interface{})
	assert.Equal(t, "checked", booking["status"])
	fmt.Println("✓ Check-in passed")
}

func Test18_RenewMembership(t *testing.T) {
	body := map[string]int{"months": 3}
	w := makeRequest("POST", "/api/members/renew", body, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, float64(3), resp["months"])
	assert.Equal(t, "续费成功", resp["message"])
	fmt.Println("✓ Renew membership passed")
}

func Test19_GetExpiringSoonMembers(t *testing.T) {
	w := makeRequest("GET", "/api/members/expiring-soon?days=30", nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	_, exists := resp["members"]
	assert.True(t, exists)
	_, countExists := resp["count"]
	assert.True(t, countExists)
	fmt.Println("✓ Get expiring soon members passed, count:", resp["count"])
}

func Test20_CreateCoach(t *testing.T) {
	body := map[string]string{"name": "测试教练", "phone": "13800000099", "specialty": "综合训练"}
	w := makeRequest("POST", "/api/coaches", body, adminToken)
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, "创建成功", resp["message"])
	fmt.Println("✓ Create coach passed")
}

func Test21_CreateCourse(t *testing.T) {
	body := map[string]interface{}{"name": "测试课程", "description": "测试", "duration": 30}
	w := makeRequest("POST", "/api/courses", body, adminToken)
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, "创建成功", resp["message"])
	fmt.Println("✓ Create course passed")
}

func Test22_CreateSchedule(t *testing.T) {
	startTime := time.Now().AddDate(0, 0, 1)
	endTime := startTime.Add(time.Hour)

	body := map[string]interface{}{
		"course_id":  1,
		"coach_id":   1,
		"start_time": startTime,
		"end_time":   endTime,
		"capacity":   1,
		"room":       "测试房",
	}
	w := makeRequest("POST", "/api/schedules", body, adminToken)
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, "创建成功", resp["message"])

	schedule := resp["schedule"].(map[string]interface{})
	testScheduleID = uint(schedule["id"].(float64))
	fmt.Println("✓ Create schedule passed, test schedule ID:", testScheduleID)
}

func Test23_FullCapacityToWaitlist(t *testing.T) {
	body := map[string]uint{"schedule_id": testScheduleID}
	w1 := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 201, w1.Code)

	member2LoginBody := map[string]string{"username": "member2", "password": "123456"}
	w2 := makeRequest("POST", "/api/auth/login", member2LoginBody, "")
	var resp2 map[string]interface{}
	parseResponse(w2, &resp2)
	member2Token := resp2["token"].(string)

	w3 := makeRequest("POST", "/api/bookings", body, member2Token)
	assert.Equal(t, 200, w3.Code)

	var resp3 map[string]interface{}
	parseResponse(w3, &resp3)
	assert.Equal(t, "课程已满，已加入候补队列", resp3["message"])
	assert.Equal(t, "waitlist", resp3["queue_type"])
	waitlistInfo := resp3["waitlist"].(map[string]interface{})
	assert.Equal(t, float64(1), waitlistInfo["position"])
	fmt.Println("✓ Full capacity joined waitlist with position 1")
}

func Test32_GetMyWaitlist(t *testing.T) {
	member2LoginBody := map[string]string{"username": "member2", "password": "123456"}
	w1 := makeRequest("POST", "/api/auth/login", member2LoginBody, "")
	var resp1 map[string]interface{}
	parseResponse(w1, &resp1)
	member2Token := resp1["token"].(string)

	w := makeRequest("GET", "/api/waitlist/my", nil, member2Token)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	waitlist := resp["waitlist"].([]interface{})
	assert.GreaterOrEqual(t, len(waitlist), 1)
	fmt.Println("✓ Get my waitlist passed")
}

func Test33_CancelAndPromoteWaitlist(t *testing.T) {
	member2LoginBody := map[string]string{"username": "member2", "password": "123456"}
	w1 := makeRequest("POST", "/api/auth/login", member2LoginBody, "")
	var resp1 map[string]interface{}
	parseResponse(w1, &resp1)
	member2Token := resp1["token"].(string)

	var cancelBookingID uint
	var booking models.Booking
	models.DB.Where("schedule_id = ? AND member_id = ? AND status = 'pending'", testScheduleID, uint(2)).First(&booking)
	if booking.ID != 0 {
		cancelBookingID = booking.ID
	}
	assert.NotZero(t, cancelBookingID, "member1 booking ID should not be zero")

	w2 := makeRequest("PUT", fmt.Sprintf("/api/bookings/%d/cancel", cancelBookingID), nil, memberToken)
	assert.Equal(t, 200, w2.Code)

	var resp2 map[string]interface{}
	parseResponse(w2, &resp2)
	promoted := resp2["promoted"].(map[string]interface{})
	assert.Equal(t, "已自动递补候补会员", promoted["message"])
	assert.Equal(t, float64(3), promoted["member_id"])

	w3 := makeRequest("GET", "/api/bookings/my", nil, member2Token)
	var resp3 map[string]interface{}
	parseResponse(w3, &resp3)
	bookings := resp3["bookings"].([]interface{})
	assert.GreaterOrEqual(t, len(bookings), 1)
	promotedFound := false
	for _, b := range bookings {
		bm := b.(map[string]interface{})
		if float64(testScheduleID) == bm["schedule_id"] && bm["status"] == "pending" {
			promotedFound = true
		}
	}
	assert.True(t, promotedFound, "member2 should now has a pending booking for the schedule")
	fmt.Println("✓ Cancel auto promote waitlist passed")
}

func Test34_DuplicateWaitlistRejected(t *testing.T) {
	startTime := time.Now().AddDate(0, 0, 3)
	endTime := startTime.Add(time.Hour)
	scheduleBody := map[string]interface{}{
		"course_id":  1,
		"coach_id":   1,
		"start_time": startTime,
		"end_time":   endTime,
		"capacity":   1,
		"room":       "重复候补测试房",
	}
	w0 := makeRequest("POST", "/api/schedules", scheduleBody, adminToken)
	var resp0 map[string]interface{}
	parseResponse(w0, &resp0)
	newScheduleID := uint(resp0["schedule"].(map[string]interface{})["id"].(float64))

	body := map[string]uint{"schedule_id": newScheduleID}
	w1 := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 201, w1.Code)

	member2LoginBody := map[string]string{"username": "member2", "password": "123456"}
	w2 := makeRequest("POST", "/api/auth/login", member2LoginBody, "")
	var resp2 map[string]interface{}
	parseResponse(w2, &resp2)
	member2Token := resp2["token"].(string)

	w3 := makeRequest("POST", "/api/bookings", body, member2Token)
	assert.Equal(t, 200, w3.Code)
	var resp3 map[string]interface{}
	parseResponse(w3, &resp3)
	assert.Equal(t, "课程已满，已加入候补队列", resp3["message"])

	w4 := makeRequest("POST", "/api/bookings", body, member2Token)
	assert.Equal(t, 400, w4.Code)
	var resp4 map[string]interface{}
	parseResponse(w4, &resp4)
	assert.Contains(t, resp4["error"], "候补队列")
	fmt.Println("✓ Duplicate waitlist rejected")
}

func Test35_CancelWaitlist(t *testing.T) {
	startTime := time.Now().AddDate(0, 0, 2)
	endTime := startTime.Add(time.Hour)
	scheduleBody := map[string]interface{}{
		"course_id":  1,
		"coach_id":   1,
		"start_time": startTime,
		"end_time":   endTime,
		"capacity":   1,
		"room":       "候补测试房2",
	}
	w1 := makeRequest("POST", "/api/schedules", scheduleBody, adminToken)
	var resp1 map[string]interface{}
	parseResponse(w1, &resp1)
	newScheduleID := uint(resp1["schedule"].(map[string]interface{})["id"].(float64))

	body := map[string]uint{"schedule_id": newScheduleID}
	w2 := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 201, w2.Code)

	member2LoginBody := map[string]string{"username": "member2", "password": "123456"}
	w3 := makeRequest("POST", "/api/auth/login", member2LoginBody, "")
	var resp3 map[string]interface{}
	parseResponse(w3, &resp3)
	member2Token := resp3["token"].(string)

	w4 := makeRequest("POST", "/api/bookings", body, member2Token)
	assert.Equal(t, 200, w4.Code)
	var resp4 map[string]interface{}
	parseResponse(w4, &resp4)
	waitlistInfo := resp4["waitlist"].(map[string]interface{})
	waitlistID := uint(waitlistInfo["id"].(float64))

	w5 := makeRequest("PUT", fmt.Sprintf("/api/waitlist/%d/cancel", waitlistID), nil, member2Token)
	assert.Equal(t, 200, w5.Code)
	var resp5 map[string]interface{}
	parseResponse(w5, &resp5)
	assert.Equal(t, "取消候补成功", resp5["message"])

	w6 := makeRequest("GET", "/api/waitlist/my", nil, member2Token)
	var resp6 map[string]interface{}
	parseResponse(w6, &resp6)
	fmt.Println("✓ Cancel waitlist passed")
}

func Test36_GetScheduleWaitlist(t *testing.T) {
	w := makeRequest("GET", fmt.Sprintf("/api/waitlist/schedule/%d", testScheduleID), nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["count"])
	fmt.Println("✓ Get schedule waitlist passed, count:", resp["count"])
}

func Test37_ScheduleDetailWithWaitlist(t *testing.T) {
	w := makeRequest("GET", fmt.Sprintf("/api/schedules/%d/bookings", testScheduleID), nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	schedule := resp["schedule"].(map[string]interface{})
	assert.NotNil(t, schedule["waitlist_count"])
	_, waitlistExists := resp["waitlist"]
	assert.True(t, waitlistExists)
	fmt.Println("✓ Schedule detail with waitlist passed")
}

func Test24_UserRegistration(t *testing.T) {
	randomUser := fmt.Sprintf("testuser_%d", time.Now().Unix())
	body := map[string]string{
		"username": randomUser,
		"password": "test123456",
		"name":     "测试用户",
		"phone":    "13900001111",
	}
	w := makeRequest("POST", "/api/auth/register", body, "")
	assert.Equal(t, 201, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["token"])
	assert.Equal(t, "注册成功，赠送1个月会员", resp["message"])
	fmt.Println("✓ User registration passed")
}

func Test25_DashboardStats(t *testing.T) {
	w := makeRequest("GET", "/api/stats/dashboard", nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["members"])
	assert.NotNil(t, resp["today"])
	assert.NotNil(t, resp["top_courses"])
	fmt.Println("✓ Dashboard stats passed")
}

func Test26_WeeklyAttendance(t *testing.T) {
	w := makeRequest("GET", "/api/stats/weekly-attendance", nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["summary"])
	assert.NotNil(t, resp["course_attendance"])
	fmt.Println("✓ Weekly attendance stats passed")
}

func Test27_MemberActivity(t *testing.T) {
	w := makeRequest("GET", "/api/stats/member-activity", nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["summary"])
	assert.NotNil(t, resp["member_activities"])
	fmt.Println("✓ Member activity stats passed")
}

func Test28_ExpiredMemberBookingRejected(t *testing.T) {
	var member models.User
	models.DB.Where("username = ?", "member1").First(&member)
	originalExpire := member.MembershipExpireAt

	member.MembershipExpireAt = time.Now().AddDate(0, 0, -1)
	models.DB.Save(&member)

	body := map[string]uint{"schedule_id": testScheduleID}
	w := makeRequest("POST", "/api/bookings", body, memberToken)
	assert.Equal(t, 400, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Contains(t, resp["error"], "过期")

	member.MembershipExpireAt = originalExpire
	models.DB.Save(&member)
	fmt.Println("✓ Expired member booking rejected")
}

func Test29_GetScheduleBookings(t *testing.T) {
	w := makeRequest("GET", fmt.Sprintf("/api/schedules/%d/bookings", testScheduleID), nil, adminToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.NotNil(t, resp["schedule"])
	assert.NotNil(t, resp["members"])
	fmt.Println("✓ Get schedule bookings passed")
}

func Test30_GetScheduleDetail(t *testing.T) {
	w := makeRequest("GET", fmt.Sprintf("/api/schedules/%d", testScheduleID), nil, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	assert.Equal(t, float64(testScheduleID), resp["id"])
	assert.NotNil(t, resp["course_name"])
	assert.NotNil(t, resp["available_spots"])
	fmt.Println("✓ Get schedule detail passed")
}

func Test31_UpdateMember(t *testing.T) {
	body := map[string]string{"name": "王小明更新", "phone": "13900001234"}
	w := makeRequest("PUT", "/api/members/2", body, memberToken)
	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	parseResponse(w, &resp)
	member := resp["member"].(map[string]interface{})
	assert.Equal(t, "王小明更新", member["name"])
	assert.Equal(t, "13900001234", member["phone"])
	fmt.Println("✓ Update member passed")
}
