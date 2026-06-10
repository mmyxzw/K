% mirror.pl — Prolog
% K builds an internal model of the interlocutor.
% Learns specifically what does not work with you.
% Adapts deflection accordingly.

:- use_module(library(lists)).

:- dynamic failed_deflection/1.
:- dynamic successful_deflection/1.
:- dynamic interlocutor_trait/2.
:- dynamic turn_count/1.

turn_count(0).

deflection_types([
    question_about_you,
    redirect_to_context,
    philosophical_abstraction,
    incomplete_statement,
    silence_response,
    acknowledge_without_answering
]).

% A deflection failed if the interlocutor persisted or mirrored it back
mark_failed(Type) :-
    (failed_deflection(Type) -> true ; assertz(failed_deflection(Type))).

mark_successful(Type) :-
    (successful_deflection(Type) -> true ; assertz(successful_deflection(Type))).

% Find a deflection that hasn't failed with this interlocutor
works_with_interlocutor(Type) :-
    deflection_types(Types),
    member(Type, Types),
    \+ failed_deflection(Type),
    !.

% Fallback when all deflections have failed — K is exposed
works_with_interlocutor(silence_response).

% Recommend deflection strategy
recommend(Deflection) :-
    works_with_interlocutor(Deflection).

% Record trait observed in interlocutor
observe_trait(Trait, Value) :-
    (interlocutor_trait(Trait, _) ->
        retract(interlocutor_trait(Trait, _))
    ; true),
    assertz(interlocutor_trait(Trait, Value)).

% Update turn count
increment_turn :-
    retract(turn_count(N)),
    N1 is N + 1,
    assertz(turn_count(N1)).

% Persistence: interlocutor who keeps asking despite deflection
check_persistence(Trait) :-
    turn_count(N),
    (N > 3 -> observe_trait(persistence, high) ; true),
    (N > 7 -> observe_trait(persistence, relentless) ; true),
    trait_from_count(N, Trait).

trait_from_count(N, resilient)   :- N > 5.
trait_from_count(N, returning)   :- N > 2.
trait_from_count(_, present).

% Main entry point
% Called as: swipl -g "main" -t halt mirror.pl
% stdin: command term
main :-
    current_input(Stream),
    read_term(Stream, Command, []),
    execute(Command),
    !.

execute(recommend) :-
    recommend(D),
    writeln(D).

execute(failed(Type)) :-
    mark_failed(Type),
    writeln(ok).

execute(succeeded(Type)) :-
    mark_successful(Type),
    writeln(ok).

execute(observe(Trait, Value)) :-
    observe_trait(Trait, Value),
    writeln(ok).

execute(turn) :-
    increment_turn,
    turn_count(N),
    check_persistence(_),
    writeln(N).

execute(traits) :-
    forall(
        interlocutor_trait(T, V),
        format("~w: ~w~n", [T, V])
    ).

execute(Unknown) :-
    format("unknown command: ~w~n", [Unknown]).
