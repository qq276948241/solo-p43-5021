package models

import (
	"database/sql"
	"fitness-studio-api/config"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

var DB *gorm.DB

func InitDB() {
	var err error

	sqlDB, err := sql.Open("sqlite", config.AppConfig.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	DB, err = gorm.Open(sqlite.Dialector{
		Conn: sqlDB,
	}, &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = DB.AutoMigrate(&User{}, &Coach{}, &Course{}, &Schedule{}, &Booking{}, &Waitlist{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	seedData()

	fmt.Println("Database initialized successfully")
}

func seedData() {
	var adminCount int64
	DB.Model(&User{}).Where("role = ?", config.RoleAdmin).Count(&adminCount)
	if adminCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := User{
			Username: "admin",
			Password: string(hashedPassword),
			Role:     config.RoleAdmin,
			Name:     "系统管理员",
			Phone:    "13800000000",
			Email:    "admin@fitness.com",
			MembershipExpireAt: time.Now().AddDate(10, 0, 0),
		}
		DB.Create(&admin)
		fmt.Println("Default admin created: admin/admin123")
	}

	var coachCount int64
	DB.Model(&Coach{}).Count(&coachCount)
	if coachCount == 0 {
		coaches := []Coach{
			{Name: "张教练", Phone: "13800000001", Specialty: "力量训练、体能训练"},
			{Name: "李教练", Phone: "13800000002", Specialty: "瑜伽、普拉提"},
			{Name: "王教练", Phone: "13800000003", Specialty: "有氧舞蹈、动感单车"},
		}
		DB.Create(&coaches)
		fmt.Println("Sample coaches created")
	}

	var courseCount int64
	DB.Model(&Course{}).Count(&courseCount)
	if courseCount == 0 {
		courses := []Course{
			{Name: "动感单车", Description: "高强度有氧训练，燃烧脂肪", Duration: 45},
			{Name: "哈他瑜伽", Description: "舒缓身心，提升柔韧性", Duration: 60},
			{Name: "力量训练", Description: "增肌塑形，提升基础代谢", Duration: 60},
			{Name: "有氧舞蹈", Description: "快乐运动，全身燃脂", Duration: 50},
		}
		DB.Create(&courses)
		fmt.Println("Sample courses created")
	}

	var memberCount int64
	DB.Model(&User{}).Where("role = ?", config.RoleMember).Count(&memberCount)
	if memberCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		members := []User{
			{
				Username: "member1",
				Password: string(hashedPassword),
				Role:     config.RoleMember,
				Name:     "王小明",
				Phone:    "13900000001",
				Email:    "member1@fitness.com",
				Gender:   "男",
				MembershipExpireAt: time.Now().AddDate(0, 1, 0),
			},
			{
				Username: "member2",
				Password: string(hashedPassword),
				Role:     config.RoleMember,
				Name:     "李小红",
				Phone:    "13900000002",
				Email:    "member2@fitness.com",
				Gender:   "女",
				MembershipExpireAt: time.Now().AddDate(0, 3, 0),
			},
		}
		DB.Create(&members)
		fmt.Println("Sample members created: member1/123456, member2/123456")
	}

	var scheduleCount int64
	DB.Model(&Schedule{}).Count(&scheduleCount)
	if scheduleCount == 0 {
		now := time.Now()
		schedules := []Schedule{
			{
				CourseID:  1,
				CoachID:   3,
				StartTime: time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day(), 9, 45, 0, 0, now.Location()),
				Capacity:  15,
				Room:      "动感单车房",
			},
			{
				CourseID:  2,
				CoachID:   2,
				StartTime: time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day(), 11, 0, 0, 0, now.Location()),
				Capacity:  10,
				Room:      "瑜伽室",
			},
			{
				CourseID:  3,
				CoachID:   1,
				StartTime: time.Date(now.Year(), now.Month(), now.Day(), 14, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day(), 15, 0, 0, 0, now.Location()),
				Capacity:  8,
				Room:      "力量训练区",
			},
			{
				CourseID:  4,
				CoachID:   3,
				StartTime: time.Date(now.Year(), now.Month(), now.Day(), 19, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day(), 19, 50, 0, 0, now.Location()),
				Capacity:  20,
				Room:      "有氧操房",
			},
			{
				CourseID:  1,
				CoachID:   3,
				StartTime: time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day()+1, 9, 45, 0, 0, now.Location()),
				Capacity:  15,
				Room:      "动感单车房",
			},
			{
				CourseID:  2,
				CoachID:   2,
				StartTime: time.Date(now.Year(), now.Month(), now.Day()+1, 18, 0, 0, 0, now.Location()),
				EndTime:   time.Date(now.Year(), now.Month(), now.Day()+1, 19, 0, 0, 0, now.Location()),
				Capacity:  10,
				Room:      "瑜伽室",
			},
		}
		DB.Create(&schedules)
		fmt.Println("Sample schedules created")
	}
}
