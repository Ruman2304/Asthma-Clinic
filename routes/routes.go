package routes

import (
	controllers "asthma-clinic/controller"
	"asthma-clinic/middleware"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App) {
	// Auth
	app.Post("/auth/register", controllers.Register)
	app.Post("/auth/login", controllers.Login)
	app.Get("/auth/me", middleware.AuthRequired, controllers.GetMe)

	// Users (protected)
	app.Get("/users", middleware.AuthRequired, controllers.GetAllUsers)
	app.Post("/users", controllers.CreateUser)

	// Emergency rooms (public)
	app.Get("/emergency/nearby", controllers.GetNearbyEmergencyRooms)

	// Tips (public)
	app.Get("/tips", controllers.GetAllTips)

	// Air Quality & GenAI
	app.Get("/air-quality", controllers.GetAirQuality)
	app.Get("/api/aqi/image", controllers.GetAQIImage)
	
	// Voice Assistant (WebSocket proxy)
	app.Use("/api/live/connect", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/api/live/connect", controllers.HandleLiveConnection())

	app.Get("/config", controllers.GetConfig)

	// Pollen
	app.Get("/pollen", controllers.GetPollenData)
}
