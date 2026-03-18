package models

type Tip struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Severity string `json:"severity"`
}
