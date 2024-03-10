package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	managers "firehoseMastodon/managers"
	models "firehoseMastodon/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sse "github.com/r3labs/sse/v2"
)

const (
	StreamerKey = "streamerKey"
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

func createClientConnections(comms *managers.Comms, deleteChannel chan uuid.UUID) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		streamerData := comms.SetConnection(deleteChannel)
		log.Printf("Connecting %s", streamerData.ConnectionId)
		ctx.Set(StreamerKey, streamerData)
		ctx.Next()
	}
}

func main() {
	cfg := GetConfig()
	models.DBSetup(cfg.DbName)

	postChannel := make(chan managers.TransferData)
	deleteChannel := make(chan uuid.UUID)

	r := gin.Default()
	r.GET("/health/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.GET("/stream/:lastTimeStamp", StreamFromSource)

	now := r.Group("/stream")
	comms := managers.NewComms(postChannel, deleteChannel)
	now.Use(createClientConnections(&comms, deleteChannel))
	now.GET("/now/", Streamer)

	go PullFromSource(postChannel, cfg.MastodonServerDomain, cfg.MastodonBearerToken)
	r.Run(cfg.Address)
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

func StreamFromSource(c *gin.Context) {

	unixTimestampStr := c.Param("lastTimeStamp")                     // Extract path parameter from context
	unixTimestamp, err := strconv.ParseInt(unixTimestampStr, 10, 64) // Parse string to int64
	uuid := uuid.New()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Unix timestamp"})
		return
	}
	var timeValue time.Time
	timeValue = time.Unix(unixTimestamp, 0) // Convert int64 to time.Time
	for {
		// Got some data!
		select {
		case <-c.Request.Context().Done():
			return
		default:
			outputData, err := models.GetLatestMessages(timeValue)
			if err != nil {
				log.Println(err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Connection Issue"})
				return
			}
			if len(outputData.OutputPosts) > 0 {
				for _, post := range outputData.OutputPosts {

					c.SSEvent("message", map[string]interface{}{
						"type": "data",
						"data": post,
					})
					// Flush the response to ensure the data is sent immediately
					c.Writer.Flush()
					log.Printf("Sending message from:%s to :%s", post.Id, uuid)
				}
				timeValue = outputData.LastTimeStamp
			}
		}
	}
}
