CC      = gcc
NASM    = nasm
NASMFLAGS = -f elf64
LDFLAGS   = -lc -no-pie -Wl,-z,noexecstack

silence: silence.o
	$(CC) -o silence silence.o $(LDFLAGS)

silence.o: silence.asm
	$(NASM) $(NASMFLAGS) -o silence.o silence.asm

clean:
	rm -f silence silence.o
