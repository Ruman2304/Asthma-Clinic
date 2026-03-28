package controller

import (
	"asthma-clinic/models"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ---- Google Places API response structs ----

type GooglePlace struct {
	PlaceID  string `json:"place_id"`
	Name     string `json:"name"`
	Vicinity string `json:"vicinity"`
	Geometry struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
	OpeningHours *struct {
		OpenNow bool `json:"open_now"`
	} `json:"opening_hours"`
	Rating float64 `json:"rating"`
}

type GooglePlacesResponse struct {
	Status  string        `json:"status"`
	Results []GooglePlace `json:"results"`
}

// ---- Haversine distance ----

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 3958.8 // Radius of Earth in miles
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// ---- Shared function to call Google Places ----

func fetchGooglePlaces(lat, lon float64, radius int, pType, keyword, apiKey string) ([]models.EmergencyRoom, error) {
	// Use rankby=distance to guarantee physically closest results are returned first
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&rankby=distance&type=%s&keyword=%s&key=%s",
		lat, lon, pType, keyword, apiKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var placesResp GooglePlacesResponse
	if err := json.Unmarshal(bodyBytes, &placesResp); err != nil {
		return nil, err
	}

	if placesResp.Status != "OK" && placesResp.Status != "ZERO_RESULTS" {
		return nil, fmt.Errorf("Google API error: %s", placesResp.Status)
	}

	var rooms []models.EmergencyRoom
	// Convert radius from meters to miles for filtering
	maxMiles := float64(radius) * 0.000621371

	for i, place := range placesResp.Results {
		plat := place.Geometry.Location.Lat
		plng := place.Geometry.Location.Lng
		dist := math.Round(haversine(lat, lon, plat, plng)*100) / 100

		// Manually filter by the requested radius!
		if dist > maxMiles {
			continue
		}

		isOpen := false
		if place.OpeningHours != nil {
			isOpen = place.OpeningHours.OpenNow
		}

		rating := place.Rating
		if rating == 0 {
			rating = 4.0
		}

		rooms = append(rooms, models.EmergencyRoom{
			ID:        i + 1,
			Name:      place.Name,
			Address:   place.Vicinity,
			Latitude:  plat,
			Longitude: plng,
			IsOpen24H: isOpen,
			Distance:  dist,
			WaitTime:  10 + (i * 7 % 50),
			Rating:    rating,
		})
	}

	return rooms, nil
}

// ---- Main handler ----

func GetNearbyEmergencyRooms(c *fiber.Ctx) error {
	latStr := c.Query("lat", "")
	lonStr := c.Query("lon", "")
	radiusStr := c.Query("radius", "15000")
	placeType := c.Query("type", "er")

	if latStr == "" || lonStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "lat and lon query parameters are required"})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid latitude"})
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid longitude"})
	}

	radius, _ := strconv.Atoi(radiusStr)
	if radius <= 0 || radius > 50000 {
		radius = 15000
	}

	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return c.JSON(fiber.Map{"count": 0, "rooms": []models.EmergencyRoom{}, "source": "no_api_key"})
	}

	// --- Search logic based on type ---
	var rooms []models.EmergencyRoom

	if placeType == "cvs" {
		rooms, err = fetchGooglePlaces(lat, lon, radius, "pharmacy", "cvs", apiKey)
	} else if placeType == "hospital" {
		rooms, err = fetchGooglePlaces(lat, lon, radius, "hospital", "hospital", apiKey)
	} else {
		// "er" or default
		rooms, err = fetchGooglePlaces(lat, lon, radius, "hospital", "emergency", apiKey)
		if err != nil || len(rooms) == 0 {
			rooms, err = fetchGooglePlaces(lat, lon, radius*2, "hospital", "medical+center", apiKey)
		}
		if err != nil || len(rooms) == 0 {
			rooms, err = fetchGooglePlaces(lat, lon, radius*2, "hospital", "clinic", apiKey)
		}
	}

	// --- If all Google calls fail, return empty with message ---
	if err != nil || len(rooms) == 0 {
		return c.JSON(fiber.Map{
			"count":   0,
			"rooms":   []models.EmergencyRoom{},
			"source":  "none",
			"message": "No hospitals found near your location. Please call 911 in an emergency.",
		})
	}

	// Sort by distance, cap at 10
	sort.Slice(rooms, func(i, j int) bool {
		return rooms[i].Distance < rooms[j].Distance
	})
	if len(rooms) > 10 {
		rooms = rooms[:10]
	}

	return c.JSON(fiber.Map{
		"count":  len(rooms),
		"rooms":  rooms,
		"source": "Google Places",
	})
}

func GetConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"google_maps_key": os.Getenv("GOOGLE_MAPS_API_KEY"),
	})
}
