import { Listener } from 'waifu-pieces'

export function createBrokerListener(event, callback) {
  class BrokerListener extends Listener {
    constructor(context, options) {
      super(context, { ...options, emitter: 'broker', event })
    }

    run(...args) {
      return callback(...args)
    }
  }
  return BrokerListener
}
