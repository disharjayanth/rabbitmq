package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Request struct {
	URL string `json:"url"`
}

type Feed struct {
	Entries []Entry `xml:"entry"`
}

type Entry struct {
	Link struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Thumbnail struct {
		URL string `xml:"url,attr"`
	} `xml:"thumbnail"`
	Title string `xml:"title"`
}

func GetFeedEntries(url string) ([]Entry, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	byteValue, _ := ioutil.ReadAll(resp.Body)
	var feed Feed
	xml.Unmarshal(byteValue, &feed)

	return feed.Entries, nil
}

func main() {
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	defer mongoClient.Disconnect(ctx)
	if err != nil {
		fmt.Println("Error connecting to mongodb server:", err)
		return
	}

	amqpConnection, err := amqp.Dial(os.Getenv("RABBITMQ_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer amqpConnection.Close()

	channelAmqp, err := amqpConnection.Channel()
	defer channelAmqp.Close()
	if err != nil {
		fmt.Println("error connecting to rabbitmq channel:", err)
		return
	}

	forever := make(chan bool)

	msgs, err := channelAmqp.Consume(
		os.Getenv("RABBITMQ_QUEUE"),
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var request Request
			json.Unmarshal(d.Body, &request)

			log.Println("RSS URL:", request.URL)

			entries, err := GetFeedEntries(request.URL)
			if err != nil {
				fmt.Println("error getting feed enteries:", err)
				return
			}

			collection := mongoClient.Database(os.Getenv("MONGO_DATABASE")).Collection("recipes")
			for _, entry := range entries {
				collection.InsertOne(ctx, bson.M{
					"title":     entry.Title,
					"thumbnail": entry.Thumbnail.URL,
					"url":       entry.Link.Href,
				})
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
