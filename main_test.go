package main

import (
	"encoding/json"
	models "firehoseMastodon/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthRoute(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var output map[string]string
	json.Unmarshal([]byte(w.Body.Bytes()), &output)
	assert.Equal(t, "ok", output["message"])
}

func TestAddingAndRetrievingOfPost(t *testing.T) {
	data := `{"content": "This is the content", "id": "112071216076336146", "created_at":"2024-03-10T11:44:05.000Z", "account":{"acct":"theMainUser"}}`

	models.DBSetup("/tmp/firehose_test.db")
	defer models.RemoveDb("/tmp/firehose_test.db")
	timeNow := time.Now()
	postStream([]byte(data))
	outputData, _ := models.GetLatestMessages(timeNow)
	assert.Equal(t, 1, len(outputData.OutputPosts))
	post := outputData.OutputPosts[0]
	assert.Equal(t, "This is the content", post.Content)
	assert.Equal(t, "112071216076336146", post.Id)
	assert.Equal(t, "theMainUser", post.AccountDetails.Acct)

}
