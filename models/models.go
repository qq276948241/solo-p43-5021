package models

import (
	"time"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password     string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:20;not null;default:member" json:"role"`
	Name         string    `gorm:"size:50;not null" json:"name"`
	Phone        string    `gorm:"size:20" json:"phone"`
	Email        string    `gorm:"size:100" json:"email"`
	Gender       string    `gorm:"size:10" json:"gender"`
	MembershipExpireAt time.Time `json:"membership_expire_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) MembershipStatus() string {
	if u.MembershipExpireAt.After(time.Now()) {
		return "active"
	}
	return "expired"
}

func (u *User) DaysUntilExpire() int {
	diff := time.Until(u.MembershipExpireAt)
	return int(diff.Hours() / 24)
}

type Coach struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	Phone     string    `gorm:"size:20" json:"phone"`
	Specialty string    `gorm:"size:100" json:"specialty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Course struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:50;not null" json:"name"`
	Description string    `gorm:"size:500" json:"description"`
	Duration    int       `gorm:"not null" json:"duration"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Schedule struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CourseID   uint      `gorm:"not null;index" json:"course_id"`
	Course     Course    `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	CoachID    uint      `gorm:"not null;index" json:"coach_id"`
	Coach      Coach     `gorm:"foreignKey:CoachID" json:"coach,omitempty"`
	StartTime  time.Time `gorm:"not null;index" json:"start_time"`
	EndTime    time.Time `gorm:"not null" json:"end_time"`
	Capacity   int       `gorm:"not null;default:10" json:"capacity"`
	Room       string    `gorm:"size:50" json:"room"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (s *Schedule) BookedCount() int {
	var count int64
	DB.Model(&Booking{}).Where("schedule_id = ? AND status != ?", s.ID, "canceled").Count(&count)
	return int(count)
}

func (s *Schedule) AvailableSpots() int {
	return s.Capacity - s.BookedCount()
}

type Booking struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ScheduleID uint      `gorm:"not null;index" json:"schedule_id"`
	Schedule   Schedule  `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	MemberID   uint      `gorm:"not null;index" json:"member_id"`
	Member     User      `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Status     string    `gorm:"size:20;not null;default:pending;index" json:"status"`
	CheckedAt  *time.Time `json:"checked_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Waitlist struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ScheduleID uint      `gorm:"not null;index" json:"schedule_id"`
	Schedule   Schedule  `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	MemberID   uint      `gorm:"not null;index" json:"member_id"`
	Member     User      `gorm:"foreignKey:MemberID" json:"member,omitempty"`
	Position   int       `gorm:"not null;default:0" json:"position"`
	Status     string    `gorm:"size:20;not null;default:waiting;index" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const (
	WaitlistStatusWaiting = "waiting"
	WaitlistStatusPromoted = "promoted"
	WaitlistStatusCanceled = "canceled"
)

func (s *Schedule) WaitlistCount() int {
	var count int64
	DB.Model(&Waitlist{}).Where("schedule_id = ? AND status = ?", s.ID, WaitlistStatusWaiting).Count(&count)
	return int(count)
}

type WeeklyAttendance struct {
	WeekStart   time.Time `json:"week_start"`
	CourseName  string    `json:"course_name"`
	TotalSlots  int       `json:"total_slots"`
	BookedCount int       `json:"booked_count"`
	CheckedCount int      `json:"checked_count"`
	AttendanceRate float64 `json:"attendance_rate"`
}

type MemberActivity struct {
	MemberID   uint   `json:"member_id"`
	MemberName string `json:"member_name"`
	BookCount  int    `json:"book_count"`
	CheckCount int    `json:"check_count"`
	ActivityScore int  `json:"activity_score"`
}
