package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/smtp"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPEmail    string
}

type EmailRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Name    string `json:"name"`
	message string `json:"message"`
}

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Errorf("Error loading .env file")
	}

	config := EmailConfig{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPEmail:    os.Getenv("SMTP_EMAIL"),
	}

	router := gin.Default()

	frontend_url := os.Getenv("FRONTEND_URL")
	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontend_url}, // Your frontend URL
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}))

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ping"})
	})

	router.POST("/send-email", func(c *gin.Context) {
		var emailReq EmailRequest
		if err := c.ShouldBindJSON(&emailReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}

		// Prepare email message
		msg := []byte(fmt.Sprintf("To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s\r\n", config.SMTPEmail,
			fmt.Sprintf("From: %v", emailReq.Name),
			fmt.Sprintf("Email: %v\nName: %v\nMessage: %v", emailReq.Email, emailReq.Name, emailReq.message)))

		// Set up authentication
		auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)

		// Connect to SMTP server with TLS
		client, err := smtp.Dial(config.SMTPHost + ":" + config.SMTPPort)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect: " + err.Error()})
			return
		}
		defer client.Close()

		// Start TLS
		if err = client.StartTLS(&tls.Config{ServerName: config.SMTPHost}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start TLS: " + err.Error()})
			return
		}

		// Authenticate
		if err = client.Auth(auth); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication failed: " + err.Error()})
			return
		}

		// Set the sender and recipient
		if err = client.Mail(emailReq.Email); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set sender: " + err.Error()})
			return
		}
		if err = client.Rcpt(config.SMTPEmail); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set recipient: " + err.Error()})
			return
		}

		// Send the email body
		w, err := client.Data()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open data connection: " + err.Error()})
			return
		}
		_, err = w.Write(msg)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write message: " + err.Error()})
			return
		}
		err = w.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close writer: " + err.Error()})
			return
		}

		// Quit the connection
		if err = client.Quit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to quit: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
	})

	router.Run(":8080")
}
