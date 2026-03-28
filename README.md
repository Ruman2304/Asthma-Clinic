# AsthmaCare — Your Intelligent Breathing Companion 🫁

AsthmaCare Clinic is a cutting-edge, patient-centric breathing assistant designed to provide 360-degree support for asthma management. By combining real-time environmental data with multimodal AI, the platform empowers patients to navigate their daily lives with confidence and receive immediate guidance during emergencies.

---

## 🚀 Advanced AI Features (Deep Dive)

### 1. Voice SOS Companion (Gemini Live API)
The crown jewel of the platform is the **Voice SOS Companion**, designed for hands-free assistance during a physical asthma attack.
- **Model**: `gemini-2.5-flash-native-audio-latest`
- **Technology**: Built using the **Gemini Multimodal Live API** via bidirectional WebSockets.
- **Natural Interaction**: The AI supports **Barge-in**; it stops speaking instantly if the user starts talking or coughing, providing a natural, human-like dialogue.
- **SOS Protocol**: Automatically guides the user through the "4x4x4" inhaler rule (4 puffs, 4 breaths each, 4 minutes wait) while maintaining a calm, reassuring tone.

### 2. Breath-Sync Soundscapes (Google Lyria)
Personalized therapeutic audio generated on-demand to aid in lung recovery and stress reduction.
- **Model**: `lyria-3-clip-preview`
- **Personalization**: Users select a mood (**🧘 Calm**, **🌧️ Rain**, **🌊 Ocean**, **🌲 Forest**, **🎧 Lo-Fi**), and Lyria composes a unique 30-second ambient track.
- **Biometric Syncing**: The dashboard features a **4-7-8 Breathing Animation** synced in real-time to the generated audio, guiding the user through the optimal therapeutic rhythm (4s Inhale, 7s Hold, 8s Exhale).

### 3. 3D City Visualizer (Nano Banana)
Transforms invisible environmental data into a compelling visual experience.
- **Model**: **Nano Banana** (Gemini 1.5 Flash / Imagen integration).
- **Function**: Interprets complex Air Quality Index (AQI) and Pollen data to generate a 15x12cm high-fidelity 3D visualization of the user's city. This helps patients "see" the air they are breathing and understand their risk levels through artistic representation.

---

## 🛠️ Core Features

-   **🌬️ Air Quality & Pollen Monitor**: Real-time tracking of PM2.5, PM10, ozone, and specific pollen types (grass, tree, weed) using Google's generative environmental APIs.
-   **🏥 Emergency ER Locator**: Instantly finds and maps the nearest medical facilities with one click.
-   **🌡️ Asthma Zones**: Real-time status tracking (Green, Yellow, Red) based on user-reported symptoms.
-   **💡 Personalized Tips**: Daily architectural and lifestyle advice for better respiratory health.

---

## 🏃 How to Run the Project

### Prerequisites
- **Go** (1.20+)
- **PostgreSQL** (Running locally or in the cloud)
- **Git**

### 1. Clone & Setup
```bash
git clone <repository-url>
cd Asthma-Clinic
go mod tidy
```

### 2. Environment Variables
Create a `.env` file in the root directory:
```env
DATABASE_URL=host=127.0.0.1 port=5432 user=postgres password=YOUR_PASSWORD dbname=Asthma_care sslmode=disable
GOOGLE_MAPS_API_KEY=YOUR_MAPS_KEY
GEMINI_API_KEY=YOUR_GEMINI_KEY
```

### 3. Database Initializaton
Ensure your PostgreSQL server is running and the database `Asthma_care` exists. The application handles connection and initialization automatically via `configuration/InitDB()`.

### 4. Build & Execute
**Note**: To bypass Windows security policies (AppLocker/Application Control), always build the binary locally instead of using `go run`.

```powershell
# Build the application
go build -o asthma_clinic.exe main.go

# Run the project
.\asthma_clinic.exe
```

### 5. Access the App
Open your browser and navigate to:
**`http://localhost:3000`**

---

## 🧪 Technical Challenges Solved
*   **Dual Audio Pipeline**: Developed a specialized logic in `live_companion.js` to handle different sample rates (16kHz for mic input vs 24kHz for AI voice output).
*   **Secure WebSocket Proxy**: Implemented a Go-based proxy to protect API keys while allowing low-latency streaming between the browser and Gemini's servers.
*   **Binary Frame Parsing**: Created a conversion layer to handle Gemini's binary WebSocket frames (Blobs) for seamless JSON interoperability.

---
*Built with ❤️ for better breathing.*
