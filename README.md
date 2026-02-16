# SME Assistant

A Docker-based AI assistant for office workers. Ask questions in plain English and the assistant performs tasks using skills (plugins).
Works on my and your machine.

## Quick Start

```bash
echo 'OPENAI_API_KEY=sk-...' > .env
mkdir -p data
docker compose up assistant -d
```

Put your `.xlsx` files in `data/`, then ask things like:

- "What is in cell B3 of sales.xlsx?"
- "Update cell A1 in budget.xlsx to 5000"
- "List the sheets in report.xlsx"

## Configuration

Set these in `.env`:

| Variable           | Default        | Description                    |
|--------------------|----------------|--------------------------------|
| `OPENAI_API_KEY`   | (required)     | Your OpenAI API key            |
| `ASSISTANT_API_KEY` | (empty)       | API key for HTTP auth (optional) |
| `GH_TOKEN`         | (empty)        | GitHub token for gh CLI        |

## Architecture

Built on [picoclaw](https://github.com/sipeed/picoclaw) — a thin Go HTTP layer wraps the picoclaw agent loop,
exposing `/api/health` and `/api/chat` endpoints. The picoclaw agent handles LLM provider integration,
tool execution, and session management.

## Home Assistant Integration

See `ha-integration/` for a custom Home Assistant conversation agent that forwards messages to the assistant.

## License

MIT
