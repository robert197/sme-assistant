"""SME Assistant conversation entity."""

from __future__ import annotations

import logging

from homeassistant.components import conversation
from homeassistant.components.conversation import ChatLog
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.helpers import intent
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.entity_platform import AddEntitiesCallback

from .const import DOMAIN, CONF_URL

_LOGGER = logging.getLogger(__name__)


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up SME Assistant conversation entity."""
    url = hass.data[DOMAIN][entry.entry_id]["url"]
    async_add_entities([SmeAssistantConversationEntity(entry, url)])


class SmeAssistantConversationEntity(conversation.ConversationEntity):
    """SME Assistant conversation agent."""

    _attr_has_entity_name = True
    _attr_name = "SME Assistant"

    def __init__(self, entry: ConfigEntry, url: str) -> None:
        """Initialize."""
        self._url = url
        self._entry = entry
        self._attr_unique_id = entry.entry_id
        self._conversation_id: str | None = None

    @property
    def supported_languages(self) -> list[str] | str:
        """Return supported languages."""
        return "*"

    async def _async_handle_message(
        self,
        user_input: conversation.ConversationInput,
        chat_log: ChatLog,
    ) -> conversation.ConversationResult:
        """Forward message to SME Assistant HTTP API."""
        conv_id = user_input.conversation_id or self._conversation_id or ""
        language = user_input.language or "en"

        try:
            session = async_get_clientsession(self.hass)
            resp = await session.post(
                f"{self._url}/api/chat",
                json={
                    "message": user_input.text,
                    "conversation_id": conv_id if conv_id else None,
                },
                timeout=120,
            )
            data = await resp.json()
        except Exception as err:
            _LOGGER.error("Error communicating with SME Assistant: %s", err)
            intent_response = intent.IntentResponse(language=language)
            intent_response.async_set_speech(
                f"Error communicating with SME Assistant: {err}"
            )
            return conversation.ConversationResult(
                response=intent_response,
                conversation_id=conv_id,
            )

        self._conversation_id = data.get("conversation_id", conv_id)

        intent_response = intent.IntentResponse(language=language)
        intent_response.async_set_speech(data.get("response", ""))

        return conversation.ConversationResult(
            response=intent_response,
            conversation_id=self._conversation_id,
        )
