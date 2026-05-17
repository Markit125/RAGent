# RAGent: Semantic Memory AI Bot

A powerful, context-aware Telegram Chatbot written in Go. This bot leverages a ReAct (Reasoning and Acting) agent architecture to interact with users, utilizing local Large Language Models (LLMs), long-term vector memory, short-term session caching, and live web search capabilities.

## 🌟 Key Features

* **Local LLM Integration:** Powered by [Ollama](https://ollama.com/), allowing the bot to run state-of-the-art models locally for both chat generation and text embeddings.
* **Long-Term Memory (RAG):** Uses [Qdrant](https://qdrant.tech/) vector database to store and retrieve past interactions, providing the bot with context over time.
* **Short-Term Memory:** Uses [Redis](https://redis.io/) to cache active chat history and session data.
* **Web Search:** Integrated with Google Custom Search API to allow the bot to search the internet for real-time information.
* **ReAct Agent Architecture:** The bot can "think" before responding and decide whether to reply directly, save information to its memory, search its memory, or search the web.
* **Clean Architecture:** Built with maintainability in mind, strictly separating domain logic, infrastructure implementations, and transport layers.

## 🛠️ Technology Stack

* **Language:** [Go 1.24+](https://golang.org/)
* **Telegram API:** [telebot.v3](https://gopkg.in/telebot.v3)
* **LLM & Embeddings:** [Ollama](https://ollama.com/)
* **Vector Database:** [Qdrant](https://qdrant.tech/)
* **Caching:** [Redis](https://redis.io/)
* **Containerization:** [Docker & Docker Compose](https://www.docker.com/)

## 🏗️ Architecture

The project follows the principles of Clean Architecture:

* `cmd/bot/` - Entry point of the application.
* `internal/domain/` - Core business logic, interfaces, and entity models (`BotAction`, `Memory`, `Message`, etc.).
* `internal/agent/` - ReAct parser logic that interprets the LLM's thoughts and commands.
* `internal/infra/` - Infrastructure implementations for external services (Ollama, Qdrant, Redis, Google Search).
* `internal/transport/` - Delivery mechanisms (Telegram bot handlers).

## 🚀 Getting Started

### Prerequisites

* [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
* Go 1.24+ (if running locally without Docker)
* Telegram Bot Token (from [@BotFather](https://t.me/botfather))
* Google Custom Search API Key and CX (Search Engine ID)

### Setup

1. **Clone the repository:**
   ```bash
   git clone <repository_url>
   cd ChatBot
   ```

2. **Configure Environment Variables:**
   Rename `.env_example` to `.env` and fill in your API keys:
   ```env
   TELEGRAM_TOKEN=your_telegram_bot_token
   GOOGLE_API_KEY=your_google_api_key
   GOOGLE_CX=your_google_search_engine_id
   ```

3. **Configure Bot Settings:**
   Adjust the configuration in `config/bot_config.json` if necessary (e.g., changing the Ollama model, Redis TTL, etc.).

4. **Launch Infrastructure:**
   Start the required services (Qdrant, Redis, Ollama) using Docker Compose:
   ```bash
   docker-compose up -d
   ```
   *(Note: If you have Ollama running locally on your host machine, you can remove the `ollama` service from `docker-compose.yml` and adjust the host URL in your `bot_config.json`)*

5. **Run the Bot:**
   ```bash
   go run cmd/bot/main.go --config config/bot_config.json
   ```

## 🧠 How the Agent Works

The bot uses a custom parsing logic (`internal/agent/parser.go`) to interpret the LLM's responses. The LLM is instructed via the `system_prompt.txt` to output its internal thoughts using specific keywords and execute actions using commands:

* `МЫСЛЬ: <text>` - Internal reasoning (hidden from the user).
* `CMD:SAVE | <text> | <tags>` - Saves information to the vector database.
* `CMD:SEARCH | <query>` - Searches the vector database for past context.
* `CMD:GOOGLE | <query>` - Performs a web search.
* Any other text is treated as a direct reply to the user.
