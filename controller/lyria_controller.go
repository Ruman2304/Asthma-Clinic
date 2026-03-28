package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
)

// GetBreathSyncAudio generates a personalized breathing soundscape using Google Lyria
func GetBreathSyncAudio(c *fiber.Ctx) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return c.Status(500).JSON(fiber.Map{"error": "GEMINI_API_KEY not set"})
	}

	mood := c.Query("mood", "calm")

	// Build prompt based on mood
	var moodDesc string
	switch mood {
	case "rain":
		moodDesc = "gentle rain on leaves with soft distant thunder, paired with slow ambient pads"
	case "ocean":
		moodDesc = "calm ocean waves lapping on a shore with soft wind chimes and ethereal synth drones"
	case "forest":
		moodDesc = "peaceful forest ambience with birdsong, gentle wind through trees, and warm acoustic guitar"
	case "lofi":
		moodDesc = "lo-fi hip hop beats with warm vinyl crackle, mellow piano chords, and soft jazz bass"
	default:
		moodDesc = "ultra calming ambient music with soft synth pads, gentle wind sounds, and warm tones"
	}

	prompt := fmt.Sprintf(
		"Create a 30-second ambient breathing meditation track. Style: %s. "+
			"The music must follow the 4-7-8 breathing pattern: "+
			"a gentle rise over 4 seconds (inhale), a sustained peaceful plateau for 7 seconds (hold), "+
			"and a slow fade/release over 8 seconds (exhale). "+
			"Tempo around 60 BPM. Very soothing, therapeutic, no sudden sounds. "+
			"Perfect for someone with asthma doing a guided breathing exercise.",
		moodDesc,
	)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/lyria-3-clip-preview:generateContent?key=%s", apiKey)

	reqBody := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]interface{}{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_modalities": []string{"AUDIO"},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to call Lyria API: " + err.Error()})
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": "Lyria API error: " + string(bodyBytes)})
	}

	// Parse the response to extract audio data
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to parse Lyria response"})
	}

	if len(result.Candidates) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "No audio generated"})
	}

	// Find the audio part
	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData.Data != "" {
			return c.JSON(fiber.Map{
				"audio_data": part.InlineData.Data,
				"mime_type":  part.InlineData.MimeType,
				"mood":       mood,
			})
		}
	}

	return c.Status(500).JSON(fiber.Map{"error": "No audio data in response"})
}
