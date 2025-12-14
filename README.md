# GridForce AI - Decentralized Compute Network (DePIN)

![Go](https://img.shields.io/badge/Go-1.24-blue) ![Docker](https://img.shields.io/badge/Docker-Enabled-blue) ![Solidity](https://img.shields.io/badge/Solidity-^0.8.0-black) ![Ethereum](https://img.shields.io/badge/Ethereum-Sepolia-gray)

**GridForce AI** is a high-performance decentralized compute marketplace that aggregates idle GPU/CPU resources from around the globe to power AI training, rendering, and scientific calculations. By connecting providers with consumers in a trustless environment, we democratize access to high-end infrastructure.

## 🏗 Architecture

The system is built on a modular, microservices-based architecture:

*   **⚡ Orchestrator (Backend)**: High-throughput Go (Gin) server managing job dispatch, WebSocket connections, and provider lifecycle.
*   **💾 Database**: PostgreSQL for persistent storage of user credits, node stats, and job history.
*   **⛏ Provider (Miner)**: Auto-scaling, cross-platform client (Go) that executes containerized workloads (Docker) and earns rewards.
*   **⛓ Blockchain & Smart Contracts**:
    *   **Network**: Ethereum Sepolia Testnet
    *   **Token**: $GRID (ERC-20)
    *   **Purpose**: Automated reward settlements and transparent transaction logging.

## 🚀 Quick Start (Docker)

Deploy the entire Orchestrator stack (Backend + DB) in minutes.

1.  **Clone the Repository**
    ```bash
    git clone https://github.com/efeakyurt/gridforce-core.git
    cd gridforce-core
    ```

2.  **Configuration**
    Copy the example configuration file and set your credentials (Blockchain keys, etc.).
    ```bash
    cp .env.example .env
    # Edit .env and supply your private keys and contract addresses
    ```

3.  **Launch Services**
    ```bash
    docker-compose up --build -d
    ```
    The Dashboard will be available at `http://localhost:8080`.

## 💻 Client Setup (Provider)

Turn your machine into a worker node and start earning $GRID.

1.  **Start the Miner**
    ```bash
    ./downloads/start_miner.sh
    ```
2.  **Follow the Prompts**
    *   Enter your Ethereum Wallet Address.
    *   The miner will connect to the Orchestrator and wait for jobs.

## 🗺 Roadmap

- [x] **MVP Completed**: Core orchestration, WebSocket protocol, and basic job dispatch.
- [x] **Blockchain Integration**: $GRID token minting and smart contract deployment.
- [ ] **Cloud Migration**: Deploying Orchestrator to AWS/GCP for global scale.
- [ ] **Mainnet Launch**: transitioning to Ethereum Mainnet/L2 (Base/Arbitrum).

---
*Built with ❤️ by the GridForce Team*
