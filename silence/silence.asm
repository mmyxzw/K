; silence.asm — x86-64 Linux NASM
; K does not respond by default. Silence is the baseline.
; The question is what breaks it.
;
; input  (stdin): "exposure_float state_string last_delta_seconds\n"
; output (stdout): "SPEAK\n" | "SILENT\n" | "WAIT\n"
;
; Logic (from spec):
;   cmp exposure, 0.3  → jl SILENT
;   cmp state, CONTROL → DEFLECT_WITH_QUESTION (→ SPEAK)
;   cmp state, FASCINATION → SPEAK_INCOMPLETE  (→ SPEAK)
;   otherwise → WAIT or SILENT based on thresholds

global main
extern scanf
extern printf
extern strcmp
extern atof
extern atoi

section .data
    fmt_scan:      db "%lf %31s %d", 0
    str_speak:     db "SPEAK", 10, 0
    str_silent:    db "SILENT", 10, 0
    str_wait:      db "WAIT", 10, 0
    str_control:   db "control", 0
    str_fascination: db "fascination", 0
    str_intercession: db "intercession", 0
    str_pragmatism:  db "pragmatism", 0
    str_active_waiting: db "active_waiting", 0
    threshold_low: dq 0.3         ; below this: always silent
    threshold_mid: dq 0.5         ; above this: K may speak from pragmatism
    threshold_high: dq 0.65       ; fascination speaks incompletely

section .bss
    exposure:     resq 1          ; double
    state_buf:    resb 32         ; string buffer
    last_delta:   resd 1          ; int (seconds)

section .text
main:
    ; prologue
    push    rbp
    mov     rbp, rsp
    sub     rsp, 16

    ; scanf("%lf %31s %d", &exposure, state_buf, &last_delta)
    lea     rdi, [rel fmt_scan]
    lea     rsi, [rel exposure]
    lea     rdx, [rel state_buf]
    lea     rcx, [rel last_delta]
    xor     eax, eax
    call    scanf

    ; --- threshold check: exposure < 0.3 → SILENT ---
    movsd   xmm0, [rel exposure]
    movsd   xmm1, [rel threshold_low]
    ucomisd xmm0, xmm1
    jb      .silent

    ; --- compare state == "control" → SPEAK (deflect with question) ---
    lea     rdi, [rel state_buf]
    lea     rsi, [rel str_control]
    call    strcmp
    test    eax, eax
    jz      .speak

    ; --- compare state == "fascination" → SPEAK (incomplete) ---
    lea     rdi, [rel state_buf]
    lea     rsi, [rel str_fascination]
    call    strcmp
    test    eax, eax
    jz      .speak

    ; --- intercession: exposure > 0.6 → SPEAK (almost) ---
    lea     rdi, [rel state_buf]
    lea     rsi, [rel str_intercession]
    call    strcmp
    test    eax, eax
    jz      .check_intercession

    ; --- pragmatism: speak if exposure is mid-range ---
    lea     rdi, [rel state_buf]
    lea     rsi, [rel str_pragmatism]
    call    strcmp
    test    eax, eax
    jz      .check_pragmatism

    ; --- active_waiting: interlocutor hasn't left, K waits ---
    lea     rdi, [rel state_buf]
    lea     rsi, [rel str_active_waiting]
    call    strcmp
    test    eax, eax
    jz      .wait

    ; default: SILENT
    jmp     .silent

.check_intercession:
    movsd   xmm0, [rel exposure]
    movsd   xmm1, [rel threshold_high]
    ucomisd xmm0, xmm1
    jae     .speak
    jmp     .wait

.check_pragmatism:
    movsd   xmm0, [rel exposure]
    movsd   xmm1, [rel threshold_mid]
    ucomisd xmm0, xmm1
    jae     .speak
    jmp     .silent

.speak:
    lea     rdi, [rel str_speak]
    xor     eax, eax
    call    printf
    jmp     .exit

.wait:
    lea     rdi, [rel str_wait]
    xor     eax, eax
    call    printf
    jmp     .exit

.silent:
    lea     rdi, [rel str_silent]
    xor     eax, eax
    call    printf

.exit:
    xor     eax, eax
    leave
    ret
