#!/usr/bin/env python3
"""
talk.py — the interface.
K does not respond by default.
Silence is what you enter into.
"""

import subprocess
import sys
import os
import time
import signal

ROUTER_BIN = os.path.join(os.path.dirname(__file__), "router", "k_router")

DIM   = "\033[2m"
RESET = "\033[0m"
BOLD  = "\033[1m"
GREY  = "\033[38;5;245m"
WHITE = "\033[38;5;255m"


def print_k(text: str):
    if not text or not text.strip():
        print(f"{DIM}  .{RESET}")
        return
    print(f"\n{WHITE}K{RESET}  ", end="", flush=True)
    for char in text:
        print(char, end="", flush=True)
        time.sleep(0.018)
    print(f"\n{RESET}")


def main():
    if not os.path.exists(ROUTER_BIN):
        print(f"{GREY}[k] router not compiled. run: make{RESET}\n")
        router_proc = None
    else:
        try:
            log_path = os.path.join(os.path.dirname(__file__), "k.log")
            log_file = open(log_path, "w")
            router_proc = subprocess.Popen(
                [ROUTER_BIN],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=log_file,
                bufsize=1,
                text=True,
            )
            time.sleep(0.3)
        except FileNotFoundError:
            print(f"{GREY}[k] router not found. run: make{RESET}\n")
            router_proc = None

    def cleanup(sig=None, frame=None):
        if router_proc:
            router_proc.terminate()
        sys.exit(0)

    signal.signal(signal.SIGINT, cleanup)
    signal.signal(signal.SIGTERM, cleanup)

    print(f"\n{BOLD}K{RESET}\n")
    print(f"{DIM}  he is here.{RESET}\n")

    while True:
        try:
            user_input = input(f"{GREY}> {RESET}").strip()
        except (EOFError, KeyboardInterrupt):
            cleanup()
            return

        if not user_input:
            continue

        if router_proc and router_proc.poll() is None:
            try:
                router_proc.stdin.write(user_input + "\n")
                router_proc.stdin.flush()
                response = router_proc.stdout.readline().rstrip("\n")
                print_k(response)
            except BrokenPipeError:
                print(f"{DIM}  .{RESET}")
        else:
            print_k("")


if __name__ == "__main__":
    main()
