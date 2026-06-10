# the_unknown.jl — Julia
#
# Runs in background every 10 minutes.
# Attempts to classify the state K cannot name.
# When state == :mystery → classification fails.
# Logs: "undefined. possibly: —"
# Never completes.

using Sockets
using Dates

REDIS_HOST = get(ENV, "REDIS_HOST", "127.0.0.1")
REDIS_PORT = parse(Int, get(ENV, "REDIS_PORT", "6379"))

# Minimal Redis client over raw TCP
function redis_cmd(sock, args::Vector{String})::String
    # RESP protocol
    cmd = "*$(length(args))\r\n"
    for a in args
        cmd *= "\$$(length(a))\r\n$a\r\n"
    end
    write(sock, cmd)
    readline(sock; keep=false)
end

function redis_get(sock, key::String)::Union{String,Nothing}
    write(sock, "*2\r\n\$3\r\nGET\r\n\$$(length(key))\r\n$key\r\n")
    header = readline(sock; keep=false)
    if startswith(header, "\$-1") || header == "\$-1"
        return nothing
    end
    if startswith(header, "\$")
        return readline(sock; keep=false)
    end
    return nothing
end

const ATTEMPT_FRAGMENTS = [
    "something between recognition and its absence",
    "the moment before the word forms — held there",
    "attention without an object",
    "a held breath that forgot it was waiting",
    "almost recognition",
    "the place where almost_love was supposed to go",
    "grief without a referent",
    "waiting that outlasted what it was waiting for",
]

function attempt_classification(state::String)::String
    # mystery and almost_love: classification always fails
    if state ∈ ("mystery", "almost_love", "")
        return "undefined. possibly: —"
    end

    # For reachable states K can nearly name
    if state ∈ ("fascination", "intercession")
        candidates = rand(ATTEMPT_FRAGMENTS, 2)
        return "undefined. possibly: $(candidates[1])"
    end

    # All others: K knows what it is, but the_unknown still runs
    # It is looking for what K cannot see in itself
    return "undefined. possibly: —"
end

function connect_redis()
    try
        sock = connect(REDIS_HOST, REDIS_PORT)
        return sock
    catch e
        return nothing
    end
end

function run()
    @info "[the_unknown] starting — classification attempt every 10 minutes"

    while true
        try
            sock = connect_redis()
            state = "indifference"

            if sock !== nothing
                try
                    result = redis_get(sock, "k:state:current")
                    if result !== nothing
                        state = result
                    end
                catch
                end
                close(sock)
            end

            result = attempt_classification(state)
            ts = Dates.format(now(), "HH:MM:SS")
            println("[$ts] the_unknown: $result")
            flush(stdout)

        catch e
            @error "[the_unknown] error" exception=e
        end

        sleep(600)  # 10 minutes
    end
end

run()
