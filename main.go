package main

import (
	"fitness-studio-api/config"
	"fitness-studio-api/handlers"
	"fitness-studio-api/middleware"
	"fitness-studio-api/models"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig()
	models.InitDB()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Fitness Studio API is running",
		})
	})

	println("Server starting on port", config.AppConfig.ServerPort)
	println("Default admin: admin / admin123")
	println("Default member: member1 / 123456, member2 / 123456")
	r.Run(config.AppConfig.ServerPort)
}
