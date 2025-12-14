#!/bin/bash
echo "=== GRIDFORCE MINER SETUP ==="
echo "Lutfen Cüzdan Adresinizi (Wallet ID) girin:"
read wallet_id
echo "Madenci baslatiliyor... (Cikmak icin Ctrl+C)"
./downloads/client -wallet "$wallet_id"