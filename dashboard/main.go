package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Recipe struct {
	Title     string `json:"title" bson:"title"`
	Thumbnail string `json:"thumbnail" bson:"thumbnail"`
	URL       string `json:"url" bson:"url"`
}

var client *mongo.Client
var ctx context.Context
var err error

func dashboardHandler(c *gin.Context) {
	cur, err := client.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes").Find(ctx, bson.M{})
	if err != nil {
		fmt.Println("Error finding all recipesRSS docs:", err)
		return
	}
	defer cur.Close(ctx)

	recipes := make([]Recipe, 0)

	for cur.Next(ctx) {
		var recipe Recipe
		cur.Decode(&recipe)
		recipes = append(recipes, recipe)
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"recipes": recipes,
	})
}

func init() {
	ctx = context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		fmt.Println("Error connecting to mongo server:", err)
		return
	}
}

func main() {
	router := gin.Default()

	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/dashboad", dashboardHandler)

	router.Run(":4000")
}
