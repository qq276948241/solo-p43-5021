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

type RenewRequest struct {
	Months int `json:"months" binding:"required,min=1"`
}

type MemberUpdateRequest struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
	Gender string `json:"gender"`
}

func RenewMembership(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentUserRole(c)

	var memberID uint
	if role == config.RoleAdmin {
		idParam := c.Param("id")
		id, err := strconv.ParseUint(idParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会员ID"})
			return
		}
		memberID = uint(id)
	} else {
		memberID = userID
	}

	var req RenewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var member models.User
	if err := models.DB.Where("id = ? AND role = ?", memberID, config.RoleMember).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会员不存在"})
		return
	}

	var newExpireAt time.Time
	if member.MembershipExpireAt.After(time.Now()) {
		newExpireAt = member.MembershipExpireAt.AddDate(0, req.Months, 0)
	} else {
		newExpireAt = time.Now().AddDate(0, req.Months, 0)
	}

	member.MembershipExpireAt = newExpireAt
	models.DB.Save(&member)

	c.JSON(http.StatusOK, gin.H{
		"message":              "续费成功",
		"months":               req.Months,
		"new_expire_at":        newExpireAt,
		"days_until_expire":    member.DaysUntilExpire(),
	})
}

func GetMemberExpiringSoon(c *gin.Context) {
	daysParam := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysParam)
	if err != nil {
		days = 7
	}

	now := time.Now()
	deadline := now.AddDate(0, 0, days)

	var members []models.User
	models.DB.Where("role = ? AND membership_expire_at >= ? AND membership_expire_at <= ?",
		config.RoleMember, now, deadline).
		Order("membership_expire_at ASC").
		Find(&members)

	var result []gin.H
	for _, m := range members {
		result = append(result, gin.H{
			"id":                   m.ID,
			"name":                 m.Name,
			"phone":                m.Phone,
			"membership_expire_at": m.MembershipExpireAt,
			"days_until_expire":    m.DaysUntilExpire(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(result),
		"members": result,
	})
}

func GetMemberList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")
	keyword := c.Query("keyword")

	offset := (page - 1) * pageSize

	query := models.DB.Model(&models.User{}).Where("role = ?", config.RoleMember)

	if keyword != "" {
		query = query.Where("name LIKE ? OR username LIKE ? OR phone LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	now := time.Now()
	if status == "active" {
		query = query.Where("membership_expire_at > ?", now)
	} else if status == "expired" {
		query = query.Where("membership_expire_at <= ?", now)
	}

	var total int64
	query.Count(&total)

	var members []models.User
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&members)

	var result []gin.H
	for _, m := range members {
		result = append(result, gin.H{
			"id":                   m.ID,
			"username":             m.Username,
			"name":                 m.Name,
			"phone":                m.Phone,
			"email":                m.Email,
			"gender":               m.Gender,
			"membership_status":    m.MembershipStatus(),
			"membership_expire_at": m.MembershipExpireAt,
			"days_until_expire":    m.DaysUntilExpire(),
			"created_at":           m.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"members":   result,
	})
}

func GetMemberDetail(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会员ID"})
		return
	}

	var member models.User
	if err := models.DB.Where("id = ? AND role = ?", uint(id), config.RoleMember).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会员不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                   member.ID,
		"username":             member.Username,
		"name":                 member.Name,
		"phone":                member.Phone,
		"email":                member.Email,
		"gender":               member.Gender,
		"membership_status":    member.MembershipStatus(),
		"membership_expire_at": member.MembershipExpireAt,
		"days_until_expire":    member.DaysUntilExpire(),
		"created_at":           member.CreatedAt,
	})
}

func UpdateMember(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	role := middleware.GetCurrentUserRole(c)

	var memberID uint
	if role == config.RoleAdmin {
		idParam := c.Param("id")
		id, err := strconv.ParseUint(idParam, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会员ID"})
			return
		}
		memberID = uint(id)
	} else {
		memberID = userID
	}

	var req MemberUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	var member models.User
	if err := models.DB.First(&member, memberID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会员不存在"})
		return
	}

	if req.Name != "" {
		member.Name = req.Name
	}
	if req.Phone != "" {
		member.Phone = req.Phone
	}
	if req.Email != "" {
		member.Email = req.Email
	}
	if req.Gender != "" {
		member.Gender = req.Gender
	}

	models.DB.Save(&member)

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"member": gin.H{
			"id":       member.ID,
			"name":     member.Name,
			"phone":    member.Phone,
			"email":    member.Email,
			"gender":   member.Gender,
		},
	})
}

func DeleteMember(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会员ID"})
		return
	}

	var member models.User
	if err := models.DB.Where("id = ? AND role = ?", uint(id), config.RoleMember).First(&member).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会员不存在"})
		return
	}

	models.DB.Delete(&member)

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}
