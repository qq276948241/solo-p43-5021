package handlers

import (
	"fitness-studio-api/config"
	"fitness-studio-api/middleware"
	"fitness-studio-api/models"
	"fitness-studio-api/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	var user models.User
	if err := models.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
			"role":     user.Role,
		},
	})
}

func GetCurrentUser(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":                   user.ID,
			"username":             user.Username,
			"name":                 user.Name,
			"phone":                user.Phone,
			"email":                user.Email,
			"gender":               user.Gender,
			"role":                 user.Role,
			"membership_status":    user.MembershipStatus(),
			"membership_expire_at": user.MembershipExpireAt,
			"days_until_expire":    user.DaysUntilExpire(),
		},
	})
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Gender   string `json:"gender"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	var existing models.User
	if err := models.DB.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user := models.User{
		Username:           req.Username,
		Password:           hashedPassword,
		Role:               config.RoleMember,
		Name:               req.Name,
		Phone:              req.Phone,
		Email:              req.Email,
		Gender:             req.Gender,
		MembershipExpireAt: time.Now().AddDate(0, 1, 0),
	}

	if err := models.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败: " + err.Error()})
		return
	}

	token, _ := utils.GenerateToken(&user)

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功，赠送1个月会员",
		"token":   token,
		"user": gin.H{
			"id":                   user.ID,
			"username":             user.Username,
			"name":                 user.Name,
			"membership_expire_at": user.MembershipExpireAt,
		},
	})
}
