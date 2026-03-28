package controller

import (
	"log"
	"os"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	gorilla "github.com/gorilla/websocket"
)

// HandleLiveConnection acts as a WebSocket proxy between the frontend and the Gemini Multimodal Live API
func HandleLiveConnection() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			log.Println("Live API error: GEMINI_API_KEY is not set")
			c.Close()
			return
		}

		// Connect to Gemini Multimodal Live API
		// v1beta is more stable for newer API keys
		geminiURL := "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key=" + apiKey

		// Log partial key to verify .env is loaded correctly (first 5 chars)
		log.Printf("Connecting to Gemini Live (v1beta) using key starting with: %s...", apiKey[:5])

		geminiConn, resp, err := gorilla.DefaultDialer.Dial(geminiURL, nil)
		if err != nil {
			if resp != nil {
				log.Printf("Gemini Handshake Error: Status=%d, Error=%v", resp.StatusCode, err)
			} else {
				log.Printf("Failed to connect to Gemini Live API (Network Error): %v", err)
			}
			c.Close()
			return
		}
		defer geminiConn.Close()

		log.Println("Successfully connected to Gemini Live API via proxy")

		// Channel to handle connection close
		done := make(chan struct{})
		var closeOnce sync.Once
		
		closeDone := func() {
			closeOnce.Do(func() {
				close(done)
			})
		}

		// Goroutine: Read from Frontend -> Send to Gemini
		go func() {
			defer closeDone()
			for {
				messageType, msg, err := c.ReadMessage()
				if err != nil {
					log.Printf("Client read error: %v", err)
					break
				}
				err = geminiConn.WriteMessage(messageType, msg)
				if err != nil {
					log.Printf("Gemini write error: %v", err)
					break
				}
			}
		}()

		// Goroutine: Read from Gemini -> Send to Frontend
		go func() {
			defer closeDone()
			for {
				messageType, msg, err := geminiConn.ReadMessage()
				if err != nil {
					log.Printf("Gemini read error: %v", err)
					break
				}
				err = c.WriteMessage(messageType, msg)
				if err != nil {
					log.Printf("Client write error: %v", err)
					break
				}
			}
		}()

		<-done
		log.Println("Live connection closed")
	})
}
