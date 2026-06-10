.PHONY: all perception state_machine exposure silence forgetting router clean

all: perception state_machine exposure silence forgetting router
	@echo ""
	@echo "K is built."
	@echo ""
	@echo "To run:"
	@echo "  redis-server --daemonize yes"
	@echo "  ollama serve &"
	@echo "  ollama pull llama3.2"
	@echo "  ./router/k_router &"
	@echo "  python3 talk.py"

# [C++] perception — TF-IDF classifier
perception:
	@echo "[building] perception (C++)..."
	@mkdir -p perception/build
	@cd perception/build && cmake .. -DCMAKE_BUILD_TYPE=Release -DCMAKE_VERBOSE_MAKEFILE=OFF > /dev/null
	@cd perception/build && make -s
	@echo "[ok] perception"

# [Go] state_machine
state_machine:
	@echo "[building] state_machine (Go)..."
	@cd state_machine && go mod tidy && go build -o state_machine ./...
	@echo "[ok] state_machine"

# [Haskell] exposure
exposure:
	@echo "[building] exposure (Haskell)..."
	@cd exposure && ghc -O2 -o exposure Main.hs 2>&1 | grep -v "^Linking\|^\\[" || true
	@echo "[ok] exposure"

# [Asm] silence
silence:
	@echo "[building] silence (x86-64 NASM)..."
	@$(MAKE) -C silence
	@echo "[ok] silence"

# [Rust] forgetting
forgetting:
	@echo "[building] forgetting (Rust)..."
	@cd forgetting && cargo build --release 2>&1 | grep -E "^error|Compiling|Finished" || true
	@echo "[ok] forgetting"

# [Go] router (k_router)
router: state_machine
	@echo "[building] router (Go)..."
	@cd router && go mod tidy && go build -o k_router ./...
	@cp router/k_router k_router
	@echo "[ok] router → ./k_router"

clean:
	@rm -rf perception/build
	@cd state_machine && rm -f state_machine
	@cd router && rm -f k_router
	@rm -f k_router
	@$(MAKE) -C silence clean
	@cd forgetting && cargo clean
	@echo "[clean]"
