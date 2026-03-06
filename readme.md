# 🎸 FretboardAI

**FretboardAI** is an AI-powered music transcription engine that converts raw guitar audio into quantized MIDI data. It uses a high-performance **Go** backend for job orchestration and a **Python/FastAPI** microservice to handle heavy AI inference.

## 🛠️ Tech Stack

- **Orchestrator:** Go (Golang)
- **AI Microservice:** Python 3.10+ (FastAPI, TensorFlow)
- **AI Models:** Spleeter (Source Separation) & Basic Pitch (Pitch Detection)
- **Blockchain (Optional):** Solana/Casper integration for decentralized payments.

---

## 🚀 Local Setup

Follow these steps to get the environment running on your machine.

### 1. Python AI Service Setup

The Python service handles the "heavy lifting"—separating stems and transcribing notes.

```bash
# Navigate to the service directory
cd python_service

# Create and activate a virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install --upgrade pip
pip install -r requirements.txt

# Start the FastAPI server
uvicorn main:app --port 8000 --reload

```

### 2. Go Backend Setup

The Go server manages the API endpoints, file uploads, and communicates with the Python service.

```bash
# Navigate to the root directory
cd fretboardAI-Go-backend

# Install Go dependencies
go mod tidy

# Run the server
go run main.go

```

---

## 📁 Project Structure

```text
.
├── fretboardAI-Go-backend/  # Go Orchestrator (Port 8080)
│   ├── main.go              # API Entry point
│   └── uploads/             # Temporary storage for raw audio
├── python_service/          # AI Microservice (Port 8000)
│   ├── main.py              # FastAPI Wrapper
│   ├── transcriber.py       # Spleeter & Basic Pitch Logic
│   └── processed/           # Output directory for MIDI & Stems
└── .gitignore               # Optimized for Go/Python projects

```

---

## 🧪 Testing the Pipeline

Once both servers are running, you can trigger a transcription using `curl`:

```bash
curl -X POST http://localhost:8080/upload \
  -F "file=@/path/to/your/guitar_solo.mp3"

```

The Go backend will forward the request to Python, which will:

1. Isolate the guitar using **Spleeter**.
2. Generate a **Quantized MIDI** file.
3. Return the note data as JSON.

---

## 🛡️ Key Features

- **Process Isolation:** Uses subprocesses to manage GPU memory effectively.
- **Polyphonic Synchronization:** Custom algorithm to handle overlapping guitar notes and chords.
- **Grid Quantization:** Automatically snaps notes to a musical grid for better readability.

---

## 🤝 Contributing

This project was built as part of a technical journey in AI and Backend engineering. Feel free to fork and submit PRs!

---

### Pro-Tip for your GitHub

Make sure you actually have a `requirements.txt` in your `python_service` folder. You can generate it by running:

```bash
pip freeze > requirements.txt

```
