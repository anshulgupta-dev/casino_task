package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"wallet_service/internal/wallet"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file", err)
	}

	dbConnStr := os.Getenv("DB_CONN_STR")
	if dbConnStr == "" {
		dbConnStr = "postgres://pam_user:pam_pass@localhost:5433/pam_db?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dbConnStr), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}

	//walletrepo

	walletRepo := wallet.NewWalletRepositoryImpl(db)
	walletService := wallet.NewService(walletRepo)

	r := gin.Default()

	r.POST("/transaction", func(c *gin.Context) {

		var req wallet.TransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := walletService.ProcessTransaction(c.Request.Context(), req)
		if err != nil {
			if err == wallet.ErrInsufficientFunds {
				c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)

	})

	r.GET("/balance/:player_id", func(c *gin.Context) {
		playerId := c.Param("player_id")
		walletType := c.DefaultQuery("type", "main")
		currency := c.DefaultQuery("currency", "USD")

		w, err := walletService.GetBalance(c.Request.Context(), playerId, walletType, currency)
		if err != nil {
			if err == wallet.ErrWalletNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"balance": w})

	})

	fmt.Println("Server started on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
