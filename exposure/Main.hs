module Main where

-- Exposure: the heart of K.
-- float 0.0 → 1.0
-- rises:  refusal, inversion, persistence
-- falls:  K successfully deflects with a question

import System.IO (hSetBuffering, stdout, BufferMode(..))
import Data.Maybe (mapMaybe)

data Event
    = RefusalOfQuestion
    | QuestionReturned
    | PersistenceDetected
    | SilenceBrokenByYou
    | KDeflectsSuccessfully
    | LongSilenceAccepted
    deriving (Show)

parseEvent :: String -> Maybe Event
parseEvent "refusal_of_question"     = Just RefusalOfQuestion
parseEvent "question_returned"       = Just QuestionReturned
parseEvent "persistence_detected"    = Just PersistenceDetected
parseEvent "silence_broken_by_you"   = Just SilenceBrokenByYou
parseEvent "k_deflects_successfully" = Just KDeflectsSuccessfully
parseEvent "long_silence_accepted"   = Just LongSilenceAccepted
parseEvent _                         = Nothing

eventDelta :: Event -> Double
eventDelta RefusalOfQuestion     =  0.08
eventDelta QuestionReturned      =  0.12
eventDelta PersistenceDetected   =  0.06
eventDelta SilenceBrokenByYou    =  0.04
eventDelta KDeflectsSuccessfully = -0.10
eventDelta LongSilenceAccepted   = -0.05

clamp :: Double -> Double
clamp x = max 0.0 (min 1.0 x)

-- stdin: "current_exposure event1 event2 ..."
-- stdout: new_exposure
main :: IO ()
main = do
    hSetBuffering stdout LineBuffering
    line <- getLine
    let parts = words line
    case parts of
        [] -> putStrLn "0.0"
        (e:rest) -> do
            let current  = read e :: Double
                events   = mapMaybe parseEvent rest
                newScore = clamp $ foldl (\acc ev -> acc + eventDelta ev) current events
            putStrLn (show newScore)
