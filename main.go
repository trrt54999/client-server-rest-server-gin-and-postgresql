package main

import (
	"net/http"
	"strings"
	"time"

	"pr5-client-server/database"
	_ "pr5-client-server/docs"
	"pr5-client-server/dto"
	"pr5-client-server/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// @title           Incident Tracker API
// @version         1.0
// @description     REST API
// @host            localhost:8080
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                Token in format: Bearer <token>
func main() {
	db := database.InitDB()
	db.AutoMigrate(&model.Incident{}, &model.User{})

	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	jwtSecret := "yakiis_super_krutiy_secretniy_kluch"

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", register(db))
		v1.POST("/auth/login", login(db, jwtSecret))
		v1.GET("/incidents", listIncidents(db))
		v1.GET("/incidents/:id", getIncident(db))
	}

	secured := v1.Group("")
	secured.Use(AuthMiddleware(jwtSecret))
	{
		secured.POST("/incidents", createIncident(db))
		secured.PATCH("/incidents/:id/status", updateIncidentStatus(db))
		secured.DELETE("/incidents/:id", deleteIncident(db))
	}

	r.Run(":8080")
}

// register godoc
// @Summary      Registration user
// @Description  Create user. Password in hash
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      dto.RegisterUserDto  true  "Password min 6 symbols"
// @Success      201    {object}  model.User
// @Failure      400    {object}  dto.ErrorResponse
// @Failure      500    {object}  dto.ErrorResponse
// @Router       /auth/register [post]
func register(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dto.RegisterUserDto
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot hashed passworld"})
			return
		}

		user := model.User{
			Email:        input.Email,
			PasswordHash: string(hashedPassword),
		}

		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User with this email already exists!"})
			return
		}

		c.JSON(http.StatusCreated, user)
	}
}

// login godoc
// @Summary      User Auth
// @Description  Check email, pass and return subcsribe jwt-token (24h termin)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input  body      dto.LoginUserDto  true  "Account credentials"
// @Success      200    {object}  dto.TokenResponse
// @Failure      400    {object}  dto.ErrorResponse
// @Failure      401    {object}  dto.ErrorResponse
// @Failure      500    {object}  dto.ErrorResponse
// @Router       /auth/login [post]
func login(db *gorm.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input dto.LoginUserDto
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user model.User
		if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect email or password ;)"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect email or password ;)"})
			return
		}

		claims := jwt.MapClaims{
			"user_id": user.ID,
			"email":   user.Email,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot generate token :("})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

// listIncidents godoc
// @Summary      Incidents list
// @Description  Return all incidents
// @Tags         incidents
// @Produce      json
// @Success      200  {array}   model.Incident
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /incidents [get]
func listIncidents(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var incidents []model.Incident
		if err := db.Find(&incidents).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, incidents)
	}
}

// getIncident godoc
// @Summary      Inciden from id
// @Description  return 1 incident
// @Tags         incidents
// @Produce      json
// @Param        id   path      int  true  "ID incident"
// @Success      200  {object}  model.Incident
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /incidents/{id} [get]
func getIncident(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var incident model.Incident
		if err := db.First(&incident, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found!"})
			return
		}
		c.JSON(http.StatusOK, incident)
	}
}

// createIncident godoc
// @Summary      Create incident
// @Description  Create incident from auth user with token
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      dto.CreateIncidentDto  true  "Incident data"
// @Success      201    {object}  model.Incident
// @Failure      400    {object}  dto.ErrorResponse
// @Failure      401    {object}  dto.ErrorResponse
// @Failure      500    {object}  dto.ErrorResponse
// @Router       /incidents [post]
func createIncident(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("userID")

		var input dto.CreateIncidentDto
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		incident := model.Incident{
			Title:       input.Title,
			Description: input.Description,
			Priority:    input.Priority,
			Status:      "open",
			UserID:      userID.(uint),
		}

		if err := db.Create(&incident).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, incident)
	}
}

// updateIncidentStatus godoc
// @Summary      Change incident status
// @Description  Update incident status (need token)
// @Tags         incidents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int                          true  "ID incident"
// @Param        input  body      dto.UpdateIncidentStatusDto  true  "New status"
// @Success      200    {object}  model.Incident
// @Failure      400    {object}  dto.ErrorResponse
// @Failure      401    {object}  dto.ErrorResponse
// @Failure      404    {object}  dto.ErrorResponse
// @Router       /incidents/{id}/status [patch]
func updateIncidentStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var incident model.Incident

		if err := db.First(&incident, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found!"})
			return
		}

		var input dto.UpdateIncidentStatusDto
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		incident.Status = input.Status
		db.Save(&incident)

		c.JSON(http.StatusOK, incident)
	}
}

// deleteIncident godoc
// @Summary      Remove incident
// @Description  Remove incident from ID (needs JWT)
// @Tags         incidents
// @Security     BearerAuth
// @Param        id   path  int  true  "ID incident"
// @Success      204  "No Content"
// @Failure      401  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /incidents/{id} [delete]
func deleteIncident(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var incident model.Incident

		if err := db.First(&incident, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found!"})
			return
		}

		db.Delete(&incident)
		c.Status(http.StatusNoContent)
	}
}

func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized access"})
			return
		}

		c.Set("userID", uint(userIDFloat))
		c.Next()
	}
}
