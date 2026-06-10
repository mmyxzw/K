// forgetting — Rust
//
// In Chaos 2, forgetting decays memory.
// In K, forgetting decays *access*.
//
// They exist — K simply cannot reach them anymore.
// The memory is intact.
// K remembers everything.
// He simply cannot get there.

use redis::AsyncCommands;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::time::{sleep, Duration};

const EMOTIONAL_STATES: &[&str] = &[
    "indifference",
    "control",
    "pragmatism",
    "active_waiting",
    "fascination",
    "intercession",
    "mystery",
    "almost_love",
];

const DECAY_THRESHOLD_SECS: u64 = 72 * 3600; // 72 hours
const ACCESS_DECAY_FACTOR:  f64 = 0.3;
const CHECK_INTERVAL_SECS:  u64 = 3600; // every hour

fn now_unix() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs()
}

async fn decay_cycle(conn: &mut redis::aio::MultiplexedConnection) -> redis::RedisResult<()> {
    let now = now_unix();

    for &state in EMOTIONAL_STATES {
        // Read last_accessed timestamp
        let last_accessed: Option<u64> = conn
            .hget("k:state:last_accessed", state)
            .await
            .ok();

        let last = last_accessed.unwrap_or(0);
        let elapsed = now.saturating_sub(last);

        if elapsed > DECAY_THRESHOLD_SECS {
            // Decay access weight — not the memory itself
            let weight_str: Option<String> = conn
                .hget("k:state:weights", state)
                .await
                .ok();

            let current_weight: f64 = weight_str
                .as_deref()
                .and_then(|s| s.parse().ok())
                .unwrap_or(1.0);

            let new_weight = (current_weight * ACCESS_DECAY_FACTOR).max(0.01);

            conn.hset::<_, _, _, ()>("k:state:weights", state, new_weight.to_string())
                .await?;

            eprintln!(
                "[forgetting] {state}: weight {current_weight:.3} → {new_weight:.3} (not accessed in {elapsed}s)"
            );
        }
    }

    Ok(())
}

#[tokio::main]
async fn main() {
    let redis_url = std::env::var("REDIS_URL")
        .unwrap_or_else(|_| "redis://127.0.0.1:6379".to_string());

    eprintln!("[forgetting] starting — decay cycle every {CHECK_INTERVAL_SECS}s");

    loop {
        match redis::Client::open(redis_url.as_str()) {
            Ok(client) => {
                match client.get_multiplexed_async_connection().await {
                    Ok(mut conn) => {
                        if let Err(e) = decay_cycle(&mut conn).await {
                            eprintln!("[forgetting] decay error: {e}");
                        }
                    }
                    Err(e) => eprintln!("[forgetting] connection error: {e}"),
                }
            }
            Err(e) => eprintln!("[forgetting] client error: {e}"),
        }

        sleep(Duration::from_secs(CHECK_INTERVAL_SECS)).await;
    }
}
