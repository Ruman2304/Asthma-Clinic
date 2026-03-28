package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ---- Google Pollen API structs ----

type PollenAPIResponse struct {
	DailyInfo []struct {
		Date struct {
			Year  int `json:"year"`
			Month int `json:"month"`
			Day   int `json:"day"`
		} `json:"date"`
		PollenTypeInfo []struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
			InSeason    bool   `json:"inSeason"`
			IndexInfo   struct {
				Code        string `json:"code"`
				DisplayName string `json:"displayName"`
				Value       int    `json:"value"`
				Category    string `json:"category"`
				Color       struct {
					Red   float64 `json:"red"`
					Green float64 `json:"green"`
					Blue  float64 `json:"blue"`
				} `json:"color"`
			} `json:"indexInfo"`
			HealthRecommendations []string `json:"healthRecommendations"`
		} `json:"pollenTypeInfo"`
		PlantInfo []struct {
			Code        string `json:"code"`
			DisplayName string `json:"displayName"`
			InSeason    bool   `json:"inSeason"`
			IndexInfo   struct {
				Code        string `json:"code"`
				DisplayName string `json:"displayName"`
				Value       int    `json:"value"`
				Category    string `json:"category"`
			} `json:"indexInfo"`
			PlantDescription struct {
				Type            string `json:"type"`
				Family          string `json:"family"`
				Season          string `json:"season"`
				SpecialColors   string `json:"specialColors"`
				SpecialShapes   string `json:"specialShapes"`
				CrossReactivity string `json:"crossReactivity"`
			} `json:"plantDescription"`
		} `json:"plantInfo"`
	} `json:"dailyInfo"`
}

func pollenColor(value int) string {
	switch {
	case value == 0:
		return "#94a3b8"
	case value <= 1:
		return "#10b981"
	case value <= 2:
		return "#84cc16"
	case value <= 3:
		return "#f59e0b"
	case value <= 4:
		return "#f97316"
	default:
		return "#ef4444"
	}
}

func pollenAsthmaRisk(category string) string {
	switch category {
	case "NONE":
		return "No pollen risk today"
	case "VERY_LOW":
		return "Very low risk — safe for most asthma patients"
	case "LOW":
		return "Low risk — generally safe outdoors"
	case "MODERATE":
		return "Moderate risk — consider taking antihistamine before going out"
	case "HIGH":
		return "High risk — limit outdoor activity, keep inhaler handy"
	case "VERY_HIGH":
		return "Very high risk — stay indoors if possible"
	default:
		return "Monitor symptoms closely"
	}
}

func GetPollenData(c *fiber.Ctx) error {
	latStr := c.Query("lat", "")
	lonStr := c.Query("lon", "")
	daysStr := c.Query("days", "3")

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

	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 5 {
		days = 3
	}

	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return c.JSON(buildPollenFallback())
	}

	url := fmt.Sprintf(
		"https://pollen.googleapis.com/v1/forecast:lookup?key=%s&location.longitude=%f&location.latitude=%f&days=%d&languageCode=en&plantsDescription=true",
		apiKey, lon, lat, days,
	)

	resp, err := http.Get(url)
	if err != nil {
		return c.JSON(buildPollenFallback())
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(buildPollenFallback())
	}

	var pollenData PollenAPIResponse
	if err := json.Unmarshal(bodyBytes, &pollenData); err != nil {
		return c.JSON(buildPollenFallback())
	}

	if len(pollenData.DailyInfo) == 0 {
		return c.JSON(buildPollenFallback())
	}

	// Build daily forecast
	var forecast []fiber.Map
	for _, day := range pollenData.DailyInfo {
		dateStr := fmt.Sprintf("%d-%02d-%02d", day.Date.Year, day.Date.Month, day.Date.Day)

		// Parse pollen types (GRASS, TREE, WEED)
		types := fiber.Map{}
		overallMax := 0
		overallCategory := "NONE"
		overallRec := ""

		for _, pt := range day.PollenTypeInfo {
			val := pt.IndexInfo.Value
			types[pt.Code] = fiber.Map{
				"name":            pt.DisplayName,
				"value":           val,
				"category":        pt.IndexInfo.Category,
				"color":           pollenColor(val),
				"in_season":       pt.InSeason,
				"recommendations": pt.HealthRecommendations,
			}
			if val > overallMax {
				overallMax = val
				overallCategory = pt.IndexInfo.Category
				if len(pt.HealthRecommendations) > 0 {
					overallRec = pt.HealthRecommendations[0]
				}
			}
		}

		// Top plants in season
		var plants []fiber.Map
		for _, pl := range day.PlantInfo {
			if pl.InSeason && pl.IndexInfo.Value > 0 {
				plants = append(plants, fiber.Map{
					"name":     pl.DisplayName,
					"code":     pl.Code,
					"value":    pl.IndexInfo.Value,
					"category": pl.IndexInfo.Category,
					"color":    pollenColor(pl.IndexInfo.Value),
					"season":   pl.PlantDescription.Season,
					"family":   pl.PlantDescription.Family,
				})
			}
		}

		forecast = append(forecast, fiber.Map{
			"date":             dateStr,
			"overall_value":    overallMax,
			"overall_category": overallCategory,
			"overall_color":    pollenColor(overallMax),
			"asthma_risk":      pollenAsthmaRisk(overallCategory),
			"recommendation":   overallRec,
			"types":            types,
			"plants":           plants,
		})
	}

	return c.JSON(fiber.Map{
		"days":     len(forecast),
		"forecast": forecast,
		"source":   "Google Pollen API",
	})
}

func buildPollenFallback() fiber.Map {
	return fiber.Map{
		"days": 3,
		"forecast": []fiber.Map{
			{
				"date":             "Today",
				"overall_value":    2,
				"overall_category": "LOW",
				"overall_color":    "#84cc16",
				"asthma_risk":      "Low risk — generally safe outdoors",
				"recommendation":   "No specific precautions needed.",
				"types": fiber.Map{
					"GRASS": fiber.Map{"name": "Grass", "value": 2, "category": "LOW", "color": "#84cc16", "in_season": true},
					"TREE":  fiber.Map{"name": "Tree", "value": 1, "category": "VERY_LOW", "color": "#10b981", "in_season": true},
					"WEED":  fiber.Map{"name": "Weed", "value": 0, "category": "NONE", "color": "#94a3b8", "in_season": false},
				},
				"plants": []fiber.Map{},
			},
		},
		"source": "fallback",
	}
}
