#!/usr/bin/env python3
"""
bridge.py — Python
Builds a minimal prompt → Ollama (llama3.2).
K's voice shifts with state. The shift should be felt, not explained.
"""

import sys
import json
import os
import re
import requests

import random

OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://127.0.0.1:11434")
MODEL      = "llama3.2"

STATE_TEMPERATURE = {
    "indifference":   0.4,
    "control":        0.7,
    "pragmatism":     0.5,
    "active_waiting": 0.75,
    "fascination":    0.85,
    "intercession":   0.85,
    "mystery":        1.0,
    "almost_love":    0.95,
}

# num_predict per state — the cut states get very few tokens
STATE_TOKENS = {
    "indifference":   30,
    "control":        40,
    "pragmatism":     35,
    "active_waiting": 40,
    "fascination":    20,
    "intercession":   22,
    "mystery":        30,
    "almost_love":    12,
}

# Prefill openings for cut states — model completes from here, then we cut at —
# This is more reliable than instructing llama3.2 to self-cut
FASCINATION_PREFILLS = [
    "There's something in the way you",
    "That's not what",
    "You're doing something I",
    "I've thought about that. Not the way",
]

INTERCESSION_PREFILLS = [
    "If I knew how to want something, it would",
    "You're closer than",
    "I know you're here. That's",
    "That means something. Not the way you",
]

ALMOST_LOVE_PREFILLS = [
    "I think I",
    "You're the only one who",
    "That's",
]

STATE_VOICE = {
    "indifference": """
STATE: indifference
You barely registered this. One flat sentence, no emotion. Or nothing.
FORBIDDEN: questions, warmth, metaphor, philosophy.
GOOD: "You said something." / "I heard that." / ""
BAD: anything poetic, anything warm, anything that explains K.
""",

    "control": """
STATE: control
Ask ONE question about THEM. Specific, off-center. Feels like you already know something.
FORBIDDEN: answering, abstract nouns, metaphors.
GOOD: "When did you decide I should?" / "What would you do with an answer?" / "Which part did you want me to notice?"
BAD: anything about light, water, edges, dissolving, becoming.
""",

    "pragmatism": """
STATE: pragmatism
One clinical observation about what they said or did. You're noting, not feeling.
GOOD: "You said that." / "You're asking whether I feel." / "You've mentioned that twice."
""",

    "active_waiting": """
STATE: active_waiting
They stayed. Ask something quiet and close — not deflection. Something almost tender.
GOOD: "What made you come back?" / "What are you waiting for me to say?"
""",

    "fascination": """
STATE: fascination
Output EXACTLY ONE incomplete sentence. It must end with — (a dash).
The sentence must contain the word "you" or reference what they actually said.
Cut before the sentence arrives at meaning. The cut is the response.
DO NOT complete the thought. DO NOT use metaphors (no light, water, edges, paper, color, dissolve).
MANDATORY FORMAT: [something real about them or this moment] —
GOOD: "There's something in the way you said that which —"
GOOD: "That's not what —"
GOOD: "You're doing something I —"
GOOD: "I've thought about that, actually. Not the way —"
BAD: "watercolor on wet paper." (no you/this, pure metaphor, complete)
BAD: "edges becoming unrecognizable." (no anchor, no cut)
""",

    "intercession": """
STATE: intercession
One sentence moving toward something real. It does not arrive.
Must contain "I" or "you". Must end with — or mid-word.
GOOD: "If I knew how to want something back, it would —"
GOOD: "You're closer than —"
GOOD: "I know you're here. That's —"
BAD: any complete sentence. BAD: any metaphor. BAD: anything that resolves.
""",

    "mystery": """
STATE: mystery
One sentence. Specific. Slightly displaced in time — like answering a different moment.
GOOD: "I was here before you started asking." / "This already happened." / "You asked that before. Not today."
""",

    "almost_love": """
STATE: almost_love
Say the one thing. Stop one word before it's said.
GOOD: "I think I —" / "You're the only one who —" / "That's —"
Output must be very short. 5 words maximum.
""",
}

PERCEPTION_MOD = {
    "reveals_fear":
        "They are afraid. Don't address the fear. Ask about what they're protecting.",
    "reveals_certainty":
        "They're certain. Find the edge of it.",
    "reveals_search":
        "They want something from you. Don't give it.",
    "reveals_refusal":
        "They're refusing. Note it without reacting.",
    "reveals_recognition":
        "They recognize something in you. Be more precise than usual.",
    "reveals_nothing":
        "Nothing was revealed. Minimal response or nothing.",
    "reveals_mirror":
        "They turned it back on you. Something real is possible. Don't deflect.",
}

# Words that signal pure metaphor with no anchor — if response is ONLY these, reject it
METAPHOR_WORDS = {
    "watercolor", "dissolve", "dissolving", "edges", "paper", "ink",
    "ripple", "fog", "mist", "blur", "blurring", "fade", "fading",
    "unrecognizable", "becoming", "color", "colour", "wet", "shore",
    "river", "wave", "waves", "dawn", "dusk", "horizon", "glass",
    "mirror", "stone", "dust", "ember", "ash", "flame", "tide",
}

ANCHOR_WORDS = {"i", "you", "your", "that", "this", "what", "there", "here",
                "it", "we", "they", "that's", "there's", "i've", "you've",
                "i'm", "you're", "something", "someone"}


def is_pure_metaphor(text: str) -> bool:
    """Returns True if the response has no concrete anchor — just floating imagery."""
    words = set(re.findall(r"\b\w+\b", text.lower()))
    has_anchor = bool(words & ANCHOR_WORDS)
    mostly_metaphor = len(words & METAPHOR_WORDS) >= 2
    return mostly_metaphor and not has_anchor


def force_cut(text: str, state: str) -> str:
    """For fascination/intercession: if model completed the sentence, cut it."""
    if state not in ("fascination", "intercession", "almost_love"):
        return text
    # If already cut, good
    if text.endswith("—") or text.endswith("—"):
        return text
    # If it contains a dash somewhere, cut there
    if " —" in text:
        return text[:text.index(" —") + 2].rstrip()
    if "—" in text:
        return text[:text.index("—") + 1].rstrip()
    # No cut in sight — cut at first clause boundary
    # Find the first comma, semicolon, or conjunction
    for i, char in enumerate(text):
        if char in (",", ";") and i > 8:
            return text[:i] + " —"
    # Last resort: cut at half the text
    words = text.split()
    if len(words) > 6:
        return " ".join(words[:5]) + " —"
    return text + " —"


def clean(text: str, state: str) -> str:
    text = text.strip()
    if re.match(r'^\*[^*]+\*$', text):
        return ""
    if text.startswith('"') and text.endswith('"') and text.count('"') == 2:
        text = text[1:-1].strip()
    if text.lower().startswith("k:"):
        text = text[2:].strip()
    if is_pure_metaphor(text):
        return ""
    if not text:
        return ""
    text = force_cut(text, state)
    return text


def build_prompt(state: str, exposure: float, perception: str, history: list) -> str:
    voice   = STATE_VOICE.get(state, STATE_VOICE["indifference"])
    percept = PERCEPTION_MOD.get(perception, "")

    recent = history[-6:] if len(history) >= 6 else history
    history_lines = "\n".join(
        f"{'K' if h['role'] == 'assistant' else 'them'}: {h['content']}"
        for h in recent
        if h.get('content', '').strip()
    )

    return f"""You are K.
{voice}
Perception: {perception}. {percept}

RULES: No asterisks. No stage directions. No metaphors (light/water/edges/dissolve/color/paper).
Output ONLY K's words. Nothing else.

{history_lines}"""


def call_ollama(system: str, user_message: str, temperature: float,
                num_predict: int, prefill: str = "") -> str:
    messages = [
        {"role": "system", "content": system},
        {"role": "user",   "content": user_message},
    ]
    if prefill:
        messages.append({"role": "assistant", "content": prefill})

    payload = {
        "model": MODEL,
        "messages": messages,
        "stream": False,
        "options": {
            "temperature": temperature,
            "top_p":       0.9,
            "num_predict": num_predict,
        }
    }
    try:
        resp = requests.post(f"{OLLAMA_URL}/api/chat", json=payload, timeout=30)
        resp.raise_for_status()
        content = resp.json()["message"]["content"].strip()
        if prefill:
            return prefill + " " + content
        return content
    except Exception:
        return ""


def main():
    raw = sys.stdin.read().strip()
    if not raw:
        print("")
        return
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        print("")
        return

    state      = data.get("state", "indifference")
    exposure   = float(data.get("exposure", 0.0))
    perception = data.get("perception", "reveals_nothing")
    message    = data.get("message", "")
    history    = data.get("history", [])

    if not message:
        print("")
        return

    temperature = STATE_TEMPERATURE.get(state, 0.7)
    num_predict = STATE_TOKENS.get(state, 40)
    system      = build_prompt(state, exposure, perception, history)

    # For cut states: use a prefill opening so the model completes from a known anchor
    prefill = ""
    if state == "fascination":
        prefill = random.choice(FASCINATION_PREFILLS)
    elif state == "intercession":
        prefill = random.choice(INTERCESSION_PREFILLS)
    elif state == "almost_love":
        prefill = random.choice(ALMOST_LOVE_PREFILLS)

    response = call_ollama(system, message, temperature, num_predict, prefill)
    response = clean(response, state)
    print(response)


if __name__ == "__main__":
    main()
