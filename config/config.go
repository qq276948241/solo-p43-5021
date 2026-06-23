package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBPath        string
	JWTSecret     string
	JWTExpireHour int
	ServerPort    string
}

var AppConfig Config

func LoadConfig() {
	AppConfig.DBPath = getEnv("DB_PATH", "./fitness.db")
	AppConfig.JWTSecret = getEnv("JWT_SECRET", "fitness-studio-secret-key-2026")
	AppConfig.JWTExpireHour, _ = strconv.Atoi(getEnv("JWT_EXPIRE_HOUR", "24"))
	AppConfig.ServerPort = getEnv("SERVER_PORT", ":8080")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

const (
	RoleAdmin  = "admin"
	RoleMember = "member"

	BookingStatusPending = "pending"
	BookingStatusChecked = "checked"
	BookingStatusCanceled = "canceled"

	MembershipStatusActive  = "active"
	MembershipStatusExpired = "expired"
)

func GetStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)
}

func GetEndOfWeek(t time.Time) time.Time {
	return GetStartOfWeek(t).AddDate(0, 0, 7).Add(-time.Nanosecond)
}
