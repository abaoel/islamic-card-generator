# islamic-card-generator

A tiny Go web service that generates shareable **Du'a & Qur'an verse cards** (1080×1350 PNG, Instagram-portrait ratio) powered by an **LLM fallback chain** (Google Gemini → Groq → OpenAI).

Describe a life situation, receive **2–3 relevant Qur'anic verses plus a short du'a (supplication)** — every field editable before you render. Design draws on Islamic art: emerald + gold palette, crescent moon and star ornament, 8-pointed Rub el Hizb (۞) at frame corners, fleuron dividers.

Deployable to **Vercel** as a serverless function — no build config needed.

## Features

- **6 themes**: Emerald (Prophetic green), Midnight (Kaaba dark + gold), Marble (Grand Mosque white), Sunset (Maghrib warmth), Desert (sand + green), Rose (soft palette).
- Custom **accent color** override in hex.
- Verses include **surah name, reference, English translation, and Latin transliteration**.
- Live preview + one-click **Download PNG**.

## LLM fallback chain

Providers are tried in this order and only added when their API key is
set in the environment:

1. **Google Gemini** — `gemini-3.6-flash` (via OpenAI-compat endpoint)
2. **Groq** — `openai/gpt-oss-120b`
3. **OpenAI** — `gpt-4o-mini`

If a provider returns a recoverable error (HTTP 429 / 5xx, quota
exhausted, rate limited, or the pinned model was deprecated), the
service automatically falls through to the next provider. Any other
error aborts. Configure at least one key to start.

## Endpoints

`POST /api/suggest` — JSON `{"situation": "..."}` → JSON `{verses:[...], dua:"..."}`.

`POST /api/card` — JSON `{situation, recipient, verses, dua, theme, accent}` → PNG.

## Run locally

```powershell
# one-time
Copy-Item .env.example .env
# paste at least one provider key into .env (GROQ_API_KEY is the easiest)

go run ./cmd/dev
# open http://localhost:3000
```

Get a free Groq key at [console.groq.com/keys](https://console.groq.com/keys).
Add Google (`GOOGLE_GENERATIVE_AI_API_KEY`) or OpenAI (`OPENAI_API_KEY`)
to enable the fallback chain.

## Deploy to Vercel

1. Push this repo to GitHub.
2. Import at [vercel.com/new](https://vercel.com/new).
3. Add at least one of `GROQ_API_KEY`, `GOOGLE_GENERATIVE_AI_API_KEY`, or `OPENAI_API_KEY` in **Settings → Environment Variables** (Production + Preview + Development). Add more than one to enable automatic fallback when a provider's quota is exhausted.
4. Deploy.

## Project layout

```
api/
  card/card.go             # POST /api/card → PNG
  suggest/suggest.go       # POST /api/suggest → JSON
ai/
  groq.go                  # OpenAI-compatible chat client + public Chat()
  fallback.go              # Google → Groq → OpenAI provider chain
dua/
  render.go                # 1080×1350 Islamic-styled renderer
  suggest.go               # AI-driven verse+du'a suggestion
cmd/dev/main.go            # local dev server (loads .env)
public/index.html          # emerald-themed playground UI
vercel.json                # per-function memory/timeout + URL rewrites
```

## Notes on translations

- English renderings come from widely-recognized translations (Sahih International, Yusuf Ali, Pickthall).
- Transliterations use common Latin-script conventions with diacritics (e.g., `Lā yukallifu Allāhu nafsan illā wus'ahā`).
- Arabic script rendering requires text shaping and RTL support beyond what Go's standard image libraries provide; a future version could integrate a shaping library and Amiri/Scheherazade fonts.

---

Built by **Emmanuel Abao** — [abaoel@gmail.com](mailto:abaoel@gmail.com)
