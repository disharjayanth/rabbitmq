package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

var channelAmqp *amqp.Channel

type Request struct {
	URL string `json:"url"`
}

func ParseHandler(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": err.Error(),
		})
		return
	}

	data, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error marshalling to JSON slice of byte:", err)
		return
	}

	if err = channelAmqp.Publish("", os.Getenv("RABBITMQ_QUEUE"), false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	}); err != nil {
		fmt.Println("error publising amqp message:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "success",
	})

}

func init() {
	amqpConnection, err := amqp.Dial(os.Getenv("RABBITMQ_URI"))
	if err != nil {
		log.Fatal(err)
	}

	channelAmqp, err = amqpConnection.Channel()
	if err != nil {
		fmt.Println("Error opening channel:", err)
	}
}

func main() {
	router := gin.Default()

	router.POST("/parse", ParseHandler)

	router.Run(":3000")
}
