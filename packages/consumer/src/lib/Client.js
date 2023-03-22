import * as dotenv from 'dotenv'
dotenv.config()

import { fileURLToPath } from 'node:url';
import { MessageBroker, Redis, Cache, ListenerStore, HealthServer, container } from 'waifu-pieces'

export function createClient(options = {}) {
  container.redis = new Redis({
    ...options.redis,
    lazyConnect: true,
    host: process.env.REDIS_HOST,
    port: process.env.REDIS_PORT,
    db: process.env.REDIS_DB,
    password: process.env.REDIS_PASSWORD
  });
  container.cache = new Cache({
    client: container.redis,
    prefix: 'w1'
  });
  container.broker = new MessageBroker({
    redis: container.redis,
    stream: process.env.BROKER_STREAM_NAME,
    block: process.env.BROKER_BLOCK ?? 5000,
    max: process.env.BROKER_MA ?? 10
  });

  container.health = new HealthServer()

  container.stores.register(new ListenerStore());
  // TODO: https://github.com/sapphiredev/pieces/pull/231
  container.stores.registerPath(fileURLToPath(new URL('..', import.meta.url)));

}

export async function loadAll() {
	await container.redis.connect();
	await container.stores.load();
	await container.broker.listen();
  container.health.listen()
}
