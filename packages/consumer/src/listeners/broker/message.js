import { createBrokerListener } from "../../lib/structures/broker-listener.js";
import { RedisMessageType } from 'waifu-pieces'

export default createBrokerListener('message', async (data) => {
  console.log(`[DEBUG] Got ${Object.keys(RedisMessageType)[data.data.type]}`)
  console.log(data.data.data)
  return data.ack()
})
