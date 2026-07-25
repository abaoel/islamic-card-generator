// Package dua renders 1080x1350 Islamic-styled cards featuring Qur'anic
// verses and a short supplication, and asks a language model to suggest
// verses+du'a given a life situation.
package dua

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abaoel/islamic-card-generator/ai"
)

// Verse is a single Qur'anic citation.
type Verse struct {
	Surah           string `json:"surah"`           // e.g. "Al-Baqarah"
	Reference       string `json:"reference"`       // e.g. "2:286"
	Translation     string `json:"translation"`     // English translation
	Transliteration string `json:"transliteration"` // Latin-script Arabic (optional)
}

// Suggestion is what the AI produces for a given life situation.
type Suggestion struct {
	Verses []Verse `json:"verses"`
	Dua    string  `json:"dua"`
}

const suggestSystem = `You are a compassionate Islamic scholar helping someone find guidance and comfort from the Qur'an.

Given the user's life situation, respond with EXACTLY one JSON object matching this shape:
{
  "verses": [
    {
      "surah": "Surah name in English (e.g., Al-Baqarah)",
      "reference": "chapter:verse (e.g., 2:286)",
      "translation": "English translation of the verse",
      "transliteration": "Latin-script transliteration with common diacritics"
    }
  ],
  "dua": "a short du'a (supplication)"
}

Rules:
- Include 2 or 3 Qur'anic verses widely recognized as fitting the situation.
- Use accessible English translations (Sahih International, Pickthall, or Yusuf Ali).
- The transliteration uses common Latin-script conventions (e.g., "Lā yukallifu Allāhu nafsan illā wus'ahā").
- The du'a must be 60 to 120 words, warm, sincere, and non-preachy. It may begin with "Ya Allah", "O Allah", or "Bismillahir-Rahmanir-Raheem".
- Do NOT include commentary, markdown, code fences, or any text outside the JSON.
- Do NOT include Arabic script — English translation + Latin transliteration only.`

// Suggest asks the language model for verses + a du'a for the given situation.
func Suggest(ctx context.Context, situation string) (Suggestion, error) {
	var s Suggestion
	if strings.TrimSpace(situation) == "" {
		return s, fmt.Errorf("situation is empty")
	}

	raw, err := ai.Chat(ctx, suggestSystem, situation, ai.ChatOptions{
		Temperature: 0.6,
		MaxTokens:   900,
		JSON:        true,
	})
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return s, fmt.Errorf("ai returned non-JSON: %w\nraw: %s", err, raw)
	}
	if len(s.Verses) == 0 || strings.TrimSpace(s.Dua) == "" {
		return s, fmt.Errorf("ai returned incomplete data: %+v", s)
	}
	return s, nil
}
