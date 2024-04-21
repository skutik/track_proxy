package api_handler

import (
	"log"
	"net/http"
	"track_proxy/requests_storage"

	"github.com/gin-gonic/gin"
)

const REQUESTS_STORAGE = "requests_storage"

func GetRequests(c *gin.Context) {
	requests_storage.Lock.Lock()
	defer requests_storage.Lock.Unlock()
	log.Println("found requests in storage:", len(requests_storage.Storage))
	c.JSON(http.StatusOK, requests_storage.Storage)
}

func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ping": "pong",
	})
}
