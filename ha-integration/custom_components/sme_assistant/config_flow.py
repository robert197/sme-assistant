"""Config flow for SME Assistant."""

from __future__ import annotations

import aiohttp
import voluptuous as vol

from homeassistant import config_entries
from homeassistant.data_entry_flow import FlowResult

from .const import DOMAIN, CONF_URL, DEFAULT_URL


class SmeAssistantConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for SME Assistant."""

    VERSION = 1

    async def async_step_user(self, user_input=None) -> FlowResult:
        """Handle the initial step."""
        errors = {}

        if user_input is not None:
            url = user_input[CONF_URL].rstrip("/")

            # Test connection
            try:
                async with aiohttp.ClientSession() as session:
                    async with session.get(
                        f"{url}/api/health",
                        timeout=aiohttp.ClientTimeout(total=5),
                    ) as resp:
                        if resp.status == 200:
                            return self.async_create_entry(
                                title="SME Assistant",
                                data={CONF_URL: url},
                            )
                        errors["base"] = "cannot_connect"
            except Exception:
                errors["base"] = "cannot_connect"

        return self.async_show_form(
            step_id="user",
            data_schema=vol.Schema(
                {
                    vol.Required(CONF_URL, default=DEFAULT_URL): str,
                }
            ),
            errors=errors,
        )
