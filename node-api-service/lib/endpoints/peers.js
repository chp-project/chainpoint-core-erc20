/* Copyright (C) 2019 Tierion
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

const restify = require('restify')
const tmRpc = require('../tendermint-rpc.js')

async function getPeersAsync(req, res, next) {
  let netResponse = await tmRpc.getNetInfoAsync()
  if (netResponse.error) {
    switch (netResponse.error.responseCode) {
      case 404:
        return { tx: null, error: new restify.NotFoundError(`Resource not found`) }
      case 409:
        return { tx: null, error: new restify.InvalidArgumentError(netResponse.error.message) }
      default:
        console.error(`RPC error communicating with Tendermint : ${netResponse.error.message}`)
        return { tx: null, error: new restify.InternalServerError('Could not query for net info') }
    }
  }

  let decodedPeers = netResponse.result.peers.map(peer => {
    let ipBytes = Buffer.from(peer.remote_ip, 'base64').slice(-4)
    return ipBytes.join('.')
  })
  res.contentType = 'application/json'
  res.send(decodedPeers)
  return next()
}

module.exports = {
  getPeersAsync: getPeersAsync
}
