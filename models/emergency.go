package models

import "time"

type EmergencyRoom struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Address   string  `json:"address"`
	Phone     string  `json:"phone"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	WaitTime  int     `json:"wait_time_minutes"`
	IsOpen24H bool    `json:"is_open_24h"`
	Rating    float64 `json:"rating"`
	Distance  float64 `json:"distance_mi,omitempty"`
}

type AsthmaTip struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Severity  string    `json:"severity"`
	CreatedAt time.Time `json:"created_at"`
}
