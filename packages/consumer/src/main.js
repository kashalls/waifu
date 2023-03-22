import { createClient, loadAll } from './lib/Client.js';
import { container } from 'waifu-pieces'

createClient()
await loadAll()

console.log(...container.stores.map((store) => `├ Loaded ${store.size.toString()} ${store.name}.`))
console.log(`├ Receiving gateway events as ${container.broker.group}:${container.broker.name}`)
console.log('Ready!')
