import * as dotenv from 'dotenv'
dotenv.config()

import { createClient, loadAll } from '#lib/Client'

createClient()
await loadAll()

console.log('Ready!')
