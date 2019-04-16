const Web3 = require('web3')
const validator = require('validator')

const web3 = new Web3(Web3.givenProvider || 'ws://localhost:8546', null, {})

module.exports = {
  NODE_ETH_REWARDS_ADDRESS: {
    type: 'input',
    name: 'NODE_ETH_REWARDS_ADDRESS',
    message: 'Enter a valid Ethereum Rewards Address:',
    validate: input => web3.utils.isAddress(input)
  },
  CORE_PUBLIC_IP_ADDRESS: {
    type: 'input',
    name: 'CORE_PUBLIC_IP_ADDRESS',
    message: "Enter your Instance's Public IP Address:",
    validate: input => {
      if (input) {
        return validator.isIP(input, 4)
      } else {
        return true
      }
    }
  },
  BITCOIN_WIF: {
      type: 'input',
      name: 'BITCOIN_WIF',
      message: 'Enter the Bitcoin private key for your hotwallet:'
  },
  INFURA_API_KEY: {
      type: 'input',
      name: 'INFURA_API_KEY',
      message: 'Enter your Infura API key (free):'
  },
  ETHERSCAN_API_KEY: {
      type: 'input',
      name: 'ETHERSCAN_API_KEY',
      message: 'Enter your Etherscan API key (free):'
  },
  INSIGHT_API_URI: {
      type: 'input',
      name: 'INSIGHT_API_URI',
      message: "Enter the full URL to your bitcoin node's Insight API:"
  },
  AUTO_REFILL_ENABLED: {
    type: 'list',
    name: 'AUTO_REFILL_ENABLED',
    message: 'Enable automatic acquisition of credit when balance reaches 0?',
    choices: [
      {
        name: 'Enable',
        value: true
      },
      {
        name: 'Disable',
        value: false
      }
    ],
    default: true
  },
  AUTO_REFILL_AMOUNT: {
    type: 'number',
    name: 'AUTO_REFILL_AMOUNT',
    message: 'Enter Auto Refill Amount - specify in number of Credits (optional: specify if auto refill is enabled)',
    default: 720,
    validate: (val, answers) => {
      if (answers['AUTO_REFILL_ENABLED'] == true) {
        return val >= 1 && val <= 8760
      } else {
        return val >= 0 && val <= 8760
      }
    }
  }
}
