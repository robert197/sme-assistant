# SME Assistant

A Docker-based AI assistant for office workers. Ask questions in plain English and the assistant performs tasks using skills (plugins).

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- An [OpenAI API key](https://platform.openai.com/api-keys)

### 1. Clone and configure

```bash
git clone https://github.com/robert197/sme-assistant.git
cd sme-assistant
cp .env.example .env
```

Edit `.env` and add your OpenAI API key:

```
OPENAI_API_KEY=sk-proj-...
```

### 2. Start the assistant

```bash
mkdir -p data
docker compose up assistant -d
```

This builds the image and starts the assistant on port 8080. The `data/` directory is mounted as the agent's workspace — put files there that you want the assistant to work with.

### 3. Verify it's running

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

### 4. Chat with the assistant

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Who are you?"}'
```

To maintain a conversation across requests, pass a `conversation_id`:

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What did I just ask?", "conversation_id": "my-session"}'
```

### 5. Start with Home Assistant (optional)

To also start the Home Assistant instance with the conversation integration:

```bash
docker compose up -d
```

Home Assistant will be available at http://localhost:8123. Add the "SME Assistant" integration from Settings > Devices & Services and point it to `http://assistant:8080`.

### 6. Stop

```bash
docker compose down
```

## Configuration

Set these in `.env`:

| Variable           | Default        | Description                    |
|--------------------|----------------|--------------------------------|
| `OPENAI_API_KEY`   | (required)     | Your OpenAI API key            |
| `ASSISTANT_API_KEY` | (empty)       | API key for HTTP auth (optional) |
| `GH_TOKEN`         | (empty)        | GitHub token for gh CLI        |

When `ASSISTANT_API_KEY` is set, all `/api/chat` requests require a `Authorization: Bearer <key>` header.

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/health` | GET | Returns `{"status":"ok"}` |
| `/api/chat` | POST | Send a message, get a response |

**POST /api/chat** body:

```json
{
  "message": "your question here",
  "conversation_id": "optional-session-id"
}
```

## Architecture

Built on [picoclaw](https://github.com/sipeed/picoclaw) — a thin Go HTTP layer ([Fiber v3](https://github.com/gofiber/fiber)) wraps the picoclaw agent loop, exposing the endpoints above. The picoclaw agent handles LLM provider integration, tool execution, and session management.

## Home Assistant Integration

See `ha-integration/` for a custom Home Assistant conversation agent that forwards messages to the assistant.

## License

MIT
