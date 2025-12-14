const hre = require("hardhat");

async function main() {
    // Kontratı hazırlıyoruz
    console.log("Deploying GridToken...");
    const gridToken = await hre.ethers.deployContract("GridToken");

    // Ağda onaylanmasını bekliyoruz
    await gridToken.waitForDeployment();

    // Adresini alıyoruz
    console.log("GridToken deployed to:", gridToken.target);
}

main().catch((error) => {
    console.error(error);
    process.exitCode = 1;
});