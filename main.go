package main

import (
	"flag"
	"log"
	"net/http"

	"encoding/json"
	"os"
	"time"

	"encoding/base64"
	managers "firehoseMastodon/managers"
	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	sse "github.com/r3labs/sse/v2"
)

// generator a function type that returns string.
type generator func() string

const (
	StreamerKey = "streamerKey"
)

var (
	random = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
)

type Account struct {
	Acct string `json:"acct"`
}
type Data struct {
	Content        string    `json:"content"`
	AccountDetails Account   `json:"account"`
	CreatedAt      time.Time `json:"created_at"`
	Id             string    `json:"id"`
}

func WithCustomHeader(key, value string) func(c *sse.Client) {
	return func(c *sse.Client) {
		c.Headers[key] = value
	}
}

func genID(len int) string {
	bytes := make([]byte, len)
	random.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)[:len]
}

func createClientConnections(comms *managers.Comms, deleteChannel chan string) gin.HandlerFunc {
	log.Println("Comms initializing.")
	return func(ctx *gin.Context) {
		postChannel := make(chan managers.TransferData)
		connectionId := genID(16)
		ok := comms.SetConnection(connectionId, postChannel)
		if !ok {
			log.Fatal("Not able to set channel")

		}
		streamerStruct := managers.StreamerData{IndividualPostChannel: postChannel, DeleteChannel: deleteChannel, ConnectionId: connectionId}
		log.Printf("Connecting %s", connectionId)
		ctx.Set(StreamerKey, streamerStruct)
		ctx.Next()
	}
}

func main() {
	var address string
	flag.StringVar(&address, "address", "0.0.0.0:7878", "port to run")
	flag.Parse()
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	mastodonServerDomain := os.Getenv("MASTODON_SERVER_DOMAIN")
	if len(mastodonServerDomain) == 0 {
		log.Fatal("domain not configured")
	}
	mastodonBearerToken := os.Getenv("MASTODON_BEARER_TOKEN")
	if len(mastodonBearerToken) == 0 {
		log.Fatal("Token not configured")
	}

	postChannel := make(chan managers.TransferData)
	deleteChannel := make(chan string)
	comms := managers.NewComms(postChannel, deleteChannel)

	go getstream(postChannel, mastodonServerDomain, mastodonBearerToken)

	r := gin.Default()

	r.Use(createClientConnections(&comms, deleteChannel))
	r.GET("/ping/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/stream", Streamer)
	r.Run(address) // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func getstream(postChannel chan<- managers.TransferData, serverDomain string, bearerToken string) {

	// add authorization header to the req
	client := sse.NewClient(serverDomain, WithCustomHeader("Authorization", bearerToken))
	log.Println("Connected to stream")
	client.Subscribe("updated", func(msg *sse.Event) {
		var streamData Data
		json.Unmarshal(msg.Data, &streamData)
		transferData := managers.TransferData{Id: streamData.Id, Data: msg.Data}
		postChannel <- transferData
	})
}

func Streamer(c *gin.Context) {
	// Get the response writer from the context
	streamerStruct, ok := c.Value(StreamerKey).(managers.StreamerData)
	if !ok {
		log.Fatal("Streamer not available in context")
	}

	for {
		select {
		case <-c.Request.Context().Done():
			streamerStruct.DeleteChannel <- streamerStruct.ConnectionId

			return
		case data, ok := <-streamerStruct.IndividualPostChannel:
			if !ok {
				// Close the response writer when the channel is closed
				return
			}
			// Got some data!
			var streamData Data
			json.Unmarshal(data.Data, &streamData)

			// Write the incoming data to the response writer
			c.SSEvent("message", map[string]interface{}{
				"type": "data",
				"data": streamData,
			})
			// Flush the response to ensure the data is sent immediately
			c.Writer.Flush()
		}
	}

}
