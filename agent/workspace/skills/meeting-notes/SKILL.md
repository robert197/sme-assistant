---
name: meeting-notes
description: Summarize meeting transcripts into structured notes with action items
---

# Meeting Notes

Turn meeting transcripts or rough notes into structured summaries.

## When to use

- User provides a meeting transcript or recording notes
- User asks to summarize meeting notes
- User asks to extract action items from a meeting
- User mentions "meeting", "minutes", "action items", or "standup"

## How to use

1. Read the transcript or notes provided by the user (file or pasted text)

2. Produce a structured summary:

```markdown
# Meeting Notes — [Topic/Title]
**Date**: [date if mentioned]
**Attendees**: [names if mentioned]

## Summary
[2-3 sentence overview of what was discussed]

## Key Decisions
- [Decision 1]
- [Decision 2]

## Action Items
- [ ] [Task] — assigned to [Person] — due [Date if mentioned]
- [ ] [Task] — assigned to [Person]

## Notes
- [Additional context or discussion points]
```

3. Save the notes to `/workspace/` if the user asks, using the format:
   `meeting-notes-YYYY-MM-DD-topic.md`

## Guidelines

- Focus on decisions and action items — those are what people need after a meeting
- Keep the summary brief; link back to the full transcript for details
- If no assignee is clear for an action item, flag it as "unassigned"
- Ask the user to confirm the notes before saving
