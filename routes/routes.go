package routes

import (
	controllers "asthma-clinic/controller"
	"asthma-clinic/middleware"

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

	// Air Quality
	app.Get("/air-quality", controllers.GetAirQuality)
	app.Get("/config", controllers.GetConfig)

	// Pollen
	app.Get("/pollen", controllers.GetPollenData)
}
