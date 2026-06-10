# the_unknown.jl — Julia
#
# Runs in background every 10 minutes.
# Attempts to classify the state K cannot name.
# Logs: "undefined. possibly: —"
# Never completes.

using Dates

const FRAGMENTS = [
    "something between recognition and its absence",
    "the moment before the word forms — held there",
    "attention without an object",
    "a held breath that forgot it was waiting",
    "almost recognition",
    "the place where almost_love was supposed to go",
    "grief without a referent",
    "waiting that outlasted what it was waiting for",
]

function read_state()::String
    state_file = joinpath(dirname(@__FILE__), "..", "k_state.tmp")
    try
        return strip(read(state_file, String))
    catch
        return "indifference"
    end
end

function attempt_classification(state::String)::String
    if state in ("mystery", "almost_love", "")
        return "undefined. possibly: —"
    end
    if state in ("fascination", "intercession")
        return "undefined. possibly: $(rand(FRAGMENTS))"
    end
    return "undefined. possibly: —"
end

function run()
    @info "[the_unknown] starting — classification attempt every 10 minutes"
    flush(stderr)

    while true
        try
            state  = read_state()
            result = attempt_classification(state)
            ts     = Dates.format(now(), "HH:MM:SS")
            println("[$ts] the_unknown: $result")
            flush(stdout)
        catch e
            # never completes. never errors loudly.
        end
        sleep(600)
    end
end

run()
