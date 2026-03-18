package controller

import (
	"asthma-clinic/configuration"
	"asthma-clinic/models"
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func GetAllTips(c *fiber.Ctx) error {
	category := c.Query("category", "")
	severity := c.Query("severity", "")

	query := `SELECT id, title, content, category, severity, created_at FROM asthma_tips WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if category != "" {
		query += fmt.Sprintf(" AND category=$%d", argIdx)
		args = append(args, category)
		argIdx++
	}
	if severity != "" {
		query += fmt.Sprintf(" AND severity=$%d", argIdx)
		args = append(args, severity)
		argIdx++
	}
	query += " ORDER BY created_at DESC"

	rows, err := configuration.DB.Query(context.Background(), query, args...)
	if err != nil {
		return c.JSON(getStaticTips())
	}
	defer rows.Close()

	var tips []models.AsthmaTip
	for rows.Next() {
		var t models.AsthmaTip
		if err := rows.Scan(&t.ID, &t.Title, &t.Content, &t.Category, &t.Severity, &t.CreatedAt); err != nil {
			continue
		}
		tips = append(tips, t)
	}
	if len(tips) == 0 {
		return c.JSON(getStaticTips())
	}
	return c.JSON(tips)
}

func getStaticTips() []models.AsthmaTip {
	return []models.AsthmaTip{
		{ID: 1, Title: "Always Carry Your Rescue Inhaler", Content: "Keep your short-acting bronchodilator inhaler with you at all times. During an asthma attack, every second counts. Store a spare at work, school, and in your car.", Category: "emergency", Severity: "critical"},
		{ID: 2, Title: "Identify Your Triggers", Content: "Common triggers include dust mites, pet dander, pollen, mold, tobacco smoke, air pollution, cold air, and exercise. Keep a diary to track what worsens your symptoms.", Category: "prevention", Severity: "moderate"},
		{ID: 3, Title: "Follow the 4-4-4 Rule During an Attack", Content: "Take 4 puffs of your reliever inhaler, wait 4 minutes, take 4 more puffs if no improvement. If still no relief after 4 more minutes, call emergency services immediately.", Category: "emergency", Severity: "critical"},
		{ID: 4, Title: "Use a Peak Flow Meter Daily", Content: "Monitoring your peak expiratory flow rate (PEFR) daily helps detect early warning signs before symptoms appear. Record readings every morning and evening.", Category: "monitoring", Severity: "moderate"},
		{ID: 5, Title: "Create an Asthma Action Plan", Content: "Work with your doctor to create a written action plan with three zones: Green (doing well), Yellow (caution), Red (medical alert). Share it with family, teachers, and coworkers.", Category: "management", Severity: "moderate"},
		{ID: 6, Title: "Proper Inhaler Technique", Content: "Shake inhaler well, breathe out fully, seal lips around mouthpiece, press while breathing in slowly for 3-5 seconds, hold breath for 10 seconds. Using a spacer improves medication delivery by 30%.", Category: "medication", Severity: "low"},
		{ID: 7, Title: "Keep Home Humidity Between 30-50%", Content: "Dust mites and mold thrive in high humidity. Use a dehumidifier, fix leaks promptly, and ventilate bathrooms. Check humidity levels with an inexpensive hygrometer.", Category: "environment", Severity: "moderate"},
		{ID: 8, Title: "Exercise Safely with Asthma", Content: "Warm up for 10-15 minutes before exercise. Consider pre-medicating with your rescue inhaler 15-20 minutes before activity if prescribed. Swimming is often recommended due to warm, humid air.", Category: "lifestyle", Severity: "low"},
		{ID: 9, Title: "Asthma Attack Warning Signs", Content: "Early signs: increased coughing (especially at night), wheezing, shortness of breath, chest tightness. Act early — don't wait until symptoms become severe to use your inhaler.", Category: "emergency", Severity: "critical"},
		{ID: 10, Title: "Avoid NSAIDs if Aspirin-Sensitive", Content: "5-10% of adults with asthma have aspirin-exacerbated respiratory disease (AERD). Avoid aspirin, ibuprofen, and naproxen. Use acetaminophen (Tylenol) for pain instead.", Category: "medication", Severity: "moderate"},
		{ID: 11, Title: "Flu & Pneumonia Vaccines Are Essential", Content: "Respiratory infections are a leading asthma trigger. Get an annual flu vaccine and discuss pneumococcal vaccine with your doctor. Asthma patients have higher risk of complications.", Category: "prevention", Severity: "moderate"},
		{ID: 12, Title: "Breathing Techniques for Asthma", Content: "Pursed-lip breathing slows breathing rate and keeps airways open longer. Diaphragmatic breathing strengthens breathing muscles. Practice daily when well — not during an attack.", Category: "lifestyle", Severity: "low"},
	}
}
