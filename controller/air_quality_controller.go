package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ---- Google Air Quality API structs ----

type GoogleAQIRequest struct {
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	ExtraComputations []string `json:"extraComputations"`
	LanguageCode      string   `json:"languageCode"`
}

type GoogleAQIResponse struct {
	DateTime string `json:"dateTime"`
	Indexes  []struct {
		Code        string `json:"code"`
		DisplayName string `json:"displayName"`
		AQI         int    `json:"aqi"`
		AQIDisplay  string `json:"aqiDisplay"`
		Color       struct {
			Red   float64 `json:"red"`
			Green float64 `json:"green"`
			Blue  float64 `json:"blue"`
		} `json:"color"`
		Category          string `json:"category"`
		DominantPollutant string `json:"dominantPollutant"`
	} `json:"indexes"`
	Pollutants []struct {
		Code          string `json:"code"`
		DisplayName   string `json:"displayName"`
		Concentration struct {
			Value float64 `json:"value"`
			Units string  `json:"units"`
		} `json:"concentration"`
	} `json:"pollutants"`
	HealthRecommendations struct {
		GeneralPopulation      string `json:"generalPopulation"`
		Elderly                string `json:"elderly"`
		LungDiseasePopulation  string `json:"lungDiseasePopulation"`
		HeartDiseasePopulation string `json:"heartDiseasePopulation"`
		Athletes               string `json:"athletes"`
		PregnantWomen          string `json:"pregnantWomen"`
		Children               string `json:"children"`
	} `json:"healthRecommendations"`
}

// ---- Google Geocoding for city name ----

type GoogleGeocodingResponse struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
	} `json:"results"`
	Status string `json:"status"`
}

func getCityFromGoogle(lat, lon float64, apiKey string) string {
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?latlng=%f,%f&result_type=locality&key=%s",
		lat, lon, apiKey,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "Your Location"
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var geoResp GoogleGeocodingResponse
	json.Unmarshal(bodyBytes, &geoResp)

	if geoResp.Status != "OK" || len(geoResp.Results) == 0 {
		return "Your Location"
	}

	city := ""
	state := ""
	for _, comp := range geoResp.Results[0].AddressComponents {
		for _, t := range comp.Types {
			if t == "locality" {
				city = comp.LongName
			}
			if t == "administrative_area_level_1" {
				state = comp.ShortName
			}
		}
	}

	if city == "" {
		return "Your Location"
	}
	if state != "" {
		return city + ", " + state
	}
	return city
}

// ---- AQI helpers ----

func aqiToScale(aqi int) int {
	// Google AQI is 0-500, convert to 1-5 scale
	switch {
	case aqi <= 50:
		return 1
	case aqi <= 100:
		return 2
	case aqi <= 150:
		return 3
	case aqi <= 200:
		return 4
	default:
		return 5
	}
}

func aqiLabel(scale int) string {
	switch scale {
	case 1:
		return "Good"
	case 2:
		return "Fair"
	case 3:
		return "Moderate"
	case 4:
		return "Poor"
	default:
		return "Very Poor"
	}
}

func aqiColor(scale int) string {
	switch scale {
	case 1:
		return "#10b981"
	case 2:
		return "#84cc16"
	case 3:
		return "#f59e0b"
	case 4:
		return "#f97316"
	default:
		return "#ef4444"
	}
}

func aqiAsthmaRisk(scale int) string {
	switch scale {
	case 1:
		return "Low risk — safe to go outside"
	case 2:
		return "Low risk — generally safe for most people"
	case 3:
		return "Moderate risk — sensitive individuals should limit outdoor activity"
	case 4:
		return "High risk — avoid prolonged outdoor activity, keep inhaler handy"
	default:
		return "Very high risk — stay indoors, use air purifier if available"
	}
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// ---- Main handler ----

func GetAirQuality(c *fiber.Ctx) error {
	latStr := c.Query("lat", "")
	lonStr := c.Query("lon", "")

	if latStr == "" || lonStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "lat and lon are required"})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid latitude"})
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid longitude"})
	}

	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return c.JSON(buildAQIFallback(lat, lon, "Your Location"))
	}

	// ---- Get city name from Google Geocoding ----
	city := getCityFromGoogle(lat, lon, apiKey)

	// ---- Call Google Air Quality API ----
	aqiReqBody := GoogleAQIRequest{
		ExtraComputations: []string{
			"HEALTH_RECOMMENDATIONS",
			"DOMINANT_POLLUTANT_CONCENTRATION",
			"POLLUTANT_CONCENTRATION",
			"LOCAL_AQI",
			"POLLUTANT_ADDITIONAL_INFO",
		},
		LanguageCode: "en",
	}
	aqiReqBody.Location.Latitude = lat
	aqiReqBody.Location.Longitude = lon

	reqBytes, _ := json.Marshal(aqiReqBody)
	aqiURL := fmt.Sprintf(
		"https://airquality.googleapis.com/v1/currentConditions:lookup?key=%s",
		apiKey,
	)

	resp, err := http.Post(aqiURL, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return c.JSON(buildAQIFallback(lat, lon, city))
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(buildAQIFallback(lat, lon, city))
	}

	var aqiData GoogleAQIResponse
	if err := json.Unmarshal(bodyBytes, &aqiData); err != nil {
		return c.JSON(buildAQIFallback(lat, lon, city))
	}

	if len(aqiData.Indexes) == 0 {
		return c.JSON(buildAQIFallback(lat, lon, city))
	}

	// ---- Parse AQI index ----
	// Use USA AQI if available, else first index
	rawAQI := aqiData.Indexes[0].AQI
	category := aqiData.Indexes[0].Category
	dominantPollutant := aqiData.Indexes[0].DominantPollutant

	for _, idx := range aqiData.Indexes {
		if idx.Code == "usa_epa" {
			rawAQI = idx.AQI
			category = idx.Category
			dominantPollutant = idx.DominantPollutant
			break
		}
	}

	scale := aqiToScale(rawAQI)

	// ---- Parse pollutants ----
	pollutants := fiber.Map{}
	for _, p := range aqiData.Pollutants {
		pollutants[p.Code] = fiber.Map{
			"value": round2(p.Concentration.Value),
			"unit":  p.Concentration.Units,
		}
	}

	// ---- Build asthma triggers ----
	triggers := []string{}
	if scale >= 3 {
		triggers = append(triggers, "Poor air quality (AQI: "+strconv.Itoa(rawAQI)+")")
	}
	if dominantPollutant == "pm25" {
		triggers = append(triggers, "PM2.5 is the dominant pollutant")
	}
	if dominantPollutant == "o3" {
		triggers = append(triggers, "High ozone levels detected")
	}
	if dominantPollutant == "no2" {
		triggers = append(triggers, "High nitrogen dioxide levels")
	}

	// ---- Health recommendations ----
	lungRec := aqiData.HealthRecommendations.LungDiseasePopulation
	if lungRec == "" {
		lungRec = aqiData.HealthRecommendations.GeneralPopulation
	}

	return c.JSON(fiber.Map{
		"city":                city,
		"aqi":                 scale,
		"aqi_raw":             rawAQI,
		"aqi_label":           aqiLabel(scale),
		"aqi_color":           aqiColor(scale),
		"category":            category,
		"dominant_pollutant":  dominantPollutant,
		"asthma_risk":         aqiAsthmaRisk(scale),
		"lung_recommendation": lungRec,
		"triggers":            triggers,
		"pollutants":          pollutants,
		"source":              "Google Air Quality API",
	})
}

func buildAQIFallback(lat, lon float64, city string) fiber.Map {
	return fiber.Map{
		"city":                city,
		"aqi":                 2,
		"aqi_raw":             75,
		"aqi_label":           "no fair",
		"aqi_color":           "#f0bc11",
		"category":            "Fair",
		"dominant_pollutant":  "pm25",
		"asthma_risk":         "Low risk — generally safe for most people",
		"lung_recommendation": "Unusually sensitive people should consider reducing prolonged outdoor exertion.",
		"triggers":            []string{},
		"pollutants": fiber.Map{
			"pm25": fiber.Map{"value": 8.0, "unit": "μg/m³"},
			"pm10": fiber.Map{"value": 12.0, "unit": "μg/m³"},
			"o3":   fiber.Map{"value": 60.0, "unit": "ppb"},
			"no2":  fiber.Map{"value": 15.0, "unit": "ppb"},
			"so2":  fiber.Map{"value": 5.0, "unit": "ppb"},
			"co":   fiber.Map{"value": 200.0, "unit": "ppb"},
		},
		"source": "fallback",
	}
}
