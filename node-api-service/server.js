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

// load all environment variables into env object
const env = require('./lib/parse-env.js')('api')

const amqp = require('amqplib')
const restify = require('restify')
const corsMiddleware = require('restify-cors-middleware')
const hashes = require('./lib/endpoints/hashes.js')
const calendar = require('./lib/endpoints/calendar.js')
const peers = require('./lib/endpoints/peers.js')
const proofs = require('./lib/endpoints/proofs.js')
const status = require('./lib/endpoints/status.js')
const root = require('./lib/endpoints/root.js')
const nodes = require('./lib/endpoints/nodes.js')
const eth = require('./lib/endpoints/eth.js')
const usageToken = require('./lib/endpoints/usage-token.js')
const connections = require('./lib/connections.js')
const proof = require('./lib/models/Proof.js')
const stakedNode = require('./lib/models/StakedNode.js')
const stakedCore = require('./lib/models/StakedCore.js')
const activeToken = require('./lib/models/ActiveToken.js')
const tmRpc = require('./lib/tendermint-rpc.js')
const tokenUtils = require('./lib/token-utils.js')
const logger = require('./lib/logger.js')
const bunyan = require('bunyan')
const apicache = require('apicache')

let redisCache = null

var apiLogs = bunyan.createLogger({
  name: 'audit',
  stream: process.stdout
})

// RESTIFY SETUP
// 'version' : all routes will default to this version
const httpOptions = {
  name: 'chainpoint',
  version: '1.0.0',
  log: apiLogs
}

const applyMiddleware = (middlewares = []) => {
  if (process.env.NODE_ENV === 'development' || process.env.NETWORK === 'testnet') {
    return []
  } else {
    return middlewares
  }
}

let throttle = (burst, rate, opts = { ip: true }) => {
  return restify.plugins.throttle(Object.assign({}, { burst, rate }, opts))
}

function setupRestifyConfigAndRoutes(server, privateMode) {
  // LOG EVERY REQUEST
  // server.pre(function (request, response, next) {
  //   request.log.info({ req: [request.url, request.method, request.rawHeaders] }, 'API-REQUEST')
  //   next()
  // })

  // Clean up sloppy paths like //todo//////1//
  server.pre(restify.plugins.pre.sanitizePath())

  // Checks whether the user agent is curl. If it is, it sets the
  // Connection header to "close" and removes the "Content-Length" header
  // See : http://restify.com/#server-api
  server.pre(restify.plugins.pre.userAgentConnection())

  // CORS
  // See : https://github.com/TabDigital/restify-cors-middleware
  // See : https://github.com/restify/node-restify/issues/1151#issuecomment-271402858
  //
  // Test w/
  //
  // curl \
  // --verbose \
  // --request OPTIONS \
  // http://127.0.0.1/hashes \
  // --header 'Origin: http://localhost:9292' \
  // --header 'Access-Control-Request-Headers: Origin, Accept, Content-Type' \
  // --header 'Access-Control-Request-Method: POST'
  //
  var cors = corsMiddleware({
    preflightMaxAge: 600,
    origins: ['*']
  })
  server.pre(cors.preflight)
  server.use(cors.actual)

  server.use(restify.plugins.gzipResponse())
  server.use(restify.plugins.queryParser())
  server.use(restify.plugins.bodyParser({ maxBodySize: env.MAX_BODY_SIZE, mapParams: true }))

  // API RESOURCES

  // submit hash(es)
  server.post({ path: '/hashes', version: '1.0.0' }, ...applyMiddleware([throttle(5, 0.02)]), hashes.postHashV1Async) // throttl
  // get the block objects for the calendar in the specified block range
  server.get(
    { path: '/calendar/:txid', version: '1.0.0' },
    ...applyMiddleware([throttle(50, 10)]),
    calendar.getCalTxAsync
  )
  // get the data value of a txId
  server.get(
    { path: '/calendar/:txid/data', version: '1.0.0' },
    ...applyMiddleware([throttle(50, 10)]),
    calendar.getCalTxDataAsync
  )
  // get proofs from storage
  if (redisCache) {
    server.get(
      { path: '/proofs', version: '1.0.0' },
      ...applyMiddleware(throttle(50, 10)),
      redisCache('1 minute'),
      proofs.getProofsByIDsAsync
    )
    logger.info('Redis caching middleware added')
  } else {
    server.get(
      { path: '/proofs', version: '1.0.0' },
      ...applyMiddleware([throttle(50, 10)]),
      proofs.getProofsByIDsAsync
    )
  }
  // get nodes from core
  server.get({ path: '/nodes/random', version: '1.0.0' }, ...applyMiddleware([throttle(15, 3)]), nodes.getNodesAsync)
  // get random core peers
  server.get({ path: '/peers', version: '1.0.0' }, ...applyMiddleware([throttle(15, 3)]), peers.getPeersAsync)
  // get status
  server.get({ path: '/status', version: '1.0.0' }, ...applyMiddleware([throttle(15, 3)]), status.getCoreStatusAsync)
  // do not enable ETH and JWT related endpoint if running in Private Mode
  if (privateMode === false) {
    // get eth tx data
    server.get(
      { path: '/eth/:addr/stats', version: '1.0.0' },
      ...applyMiddleware([throttle(5, 1)]),
      eth.getEthStatsAsync
    )
    // post eth broadcast
    server.post(
      { path: '/eth/broadcast', version: '1.0.0' },
      ...applyMiddleware([throttle(5, 1)]),
      eth.postEthBroadcastAsync
    )
    // post token refresh
    server.post({ path: '/usagetoken/refresh', version: '1.0.0' }, usageToken.postTokenRefreshAsync)
    // post token credit
    server.post({ path: '/usagetoken/credit', version: '1.0.0' }, usageToken.postTokenCreditAsync)
    // post token audience update
    server.post({ path: '/usagetoken/audience', version: '1.0.0' }, usageToken.postTokenAudienceUpdateAsync)
  }
  // teapot
  server.get({ path: '/', version: '1.0.0' }, root.getV1)
}

// HTTP Server
async function startInsecureRestifyServerAsync(privateMode) {
  let restifyServer = restify.createServer(httpOptions)
  setupRestifyConfigAndRoutes(restifyServer, privateMode)

  // Begin listening for requests
  await connections.listenRestifyAsync(restifyServer, 8080)
  return restifyServer
}

/**
 * Opens a Redis connection
 *
 * @param {string} redisURI - The connection string for the Redis instance, an Redis URI
 */
function openRedisConnection(redisURIs) {
  return new Promise(resolve => {
    connections.openRedisConnection(
      redisURIs,
      newRedis => {
        resolve(newRedis)
        tokenUtils.setRedis(newRedis)
        redisCache = apicache.options({
          redisClient: newRedis,
          debug: true,
          appendKey: req => req.headers.hashids
        }).middleware
      },
      () => {
        tokenUtils.setRedis(null)
        setTimeout(() => {
          openRedisConnection(redisURIs).then(() => resolve())
        }, 5000)
      }
    )
  })
}

/**
 * Opens a Postgres connection
 **/
async function openPostgresConnectionAsync() {
  let sqlzModelArray = [proof, stakedNode, activeToken, stakedCore]
  let cxObjects = await connections.openPostgresConnectionAsync(sqlzModelArray)
  proof.setDatabase(cxObjects.sequelize, cxObjects.op, cxObjects.models[0])
  stakedNode.setDatabase(cxObjects.sequelize, cxObjects.models[1])
  activeToken.setDatabase(cxObjects.sequelize, cxObjects.models[2])
  stakedCore.setDatabase(cxObjects.sequelize, cxObjects.models[3])
}

/**
 * Opens a Tendermint connection
 **/
async function openTendermintConnectionAsync() {
  let rpcClient = await connections.openTendermintConnectionAsync(env.TENDERMINT_URI)
  tmRpc.setRpcClient(rpcClient)
  tokenUtils.setTMRPC(tmRpc)
}

/**
 * Opens an AMPQ connection and channel
 * Retry logic is included to handle losses of connection    await openRedisConnection(env.REDIS_CONNECT_URIS)

 *
 * @param {string} connectURI - The connection URI for the RabbitMQ instance
 */
async function openRMQConnectionAsync(connectURI) {
  await connections.openStandardRMQConnectionAsync(
    amqp,
    connectURI,
    [env.RMQ_WORK_OUT_AGG_QUEUE],
    null,
    null,
    chan => {
      hashes.setAMQPChannel(chan)
    },
    () => {
      hashes.setAMQPChannel(null)
      setTimeout(() => {
        openRMQConnectionAsync(connectURI)
      }, 5000)
    }
  )
}

// process all steps need to start the application
async function start() {
  if (env.NODE_ENV === 'test') return
  if (env.PRIVATE_NETWORK) logger.info(`*** Private Network Mode ***`)
  try {
    // init Redis
    await openRedisConnection(env.REDIS_CONNECT_URIS)
    // init DB
    await openPostgresConnectionAsync()
    // init Tendermint
    await openTendermintConnectionAsync()
    // init RabbitMQ
    await openRMQConnectionAsync(env.RABBITMQ_CONNECT_URI)
    // Init Restify
    await startInsecureRestifyServerAsync(env.PRIVATE_NETWORK)
    logger.info(`Startup completed successfully ${env.PRIVATE_NETWORK ? ': *** Private Network Mode ***' : ''}`)
  } catch (error) {
    logger.error(`An error has occurred on startup : ${error.message}`)
    process.exit(1)
  }
}

// get the whole show started
start()

// export these functions for testing purposes
module.exports = {
  setAMQPChannel: chan => {
    hashes.setAMQPChannel(chan)
  },
  // additional functions for testing purposes
  startInsecureRestifyServerAsync: startInsecureRestifyServerAsync,
  setThrottle: t => {
    throttle = t
  }
}
