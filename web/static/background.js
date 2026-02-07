let t = 0;

function setup() {
    let canvas = createCanvas(windowWidth, windowHeight);
    canvas.position(0, 0);
    canvas.style('z-index', '-1');
    canvas.style('position', 'fixed');
    canvas.style('top', '0');
    canvas.style('left', '0');
}

function draw() {
    background(15, 23, 42); // Match CSS --bg-color #0f172a

    // Use the user's mesmerizing logic adapted for full screen
    // Original: stroke(w, 46) -> white with alpha
    stroke(100, 150, 255, 46); // Blue-ish tint to match theme

    t += PI / 30; // Slightly faster fluid speed

    for (let i = 15000; i > 0; i -= 2) { // Reduce count slightly for performance on large screens
        let k = i < 15000 ? sin(i / 9) * 9 : 4 * cos(i / 49) * cos(i / 3690);
        let e = i / 984 - 12;
        let d = mag(k, e) ** 2 / 99 + 1;

        let q = k * (4 + sin(d * 16 - t + k)) - 5 * sin(atan2(k, e) * 9);
        let c = d * 1.1 - t / 18 + (i % 2) * 3;

        // Scale and center
        let x = (q + 60 * sin(c)) * 1.5; // Scale up a bit
        let y = ((q + 40) * sin(c - d) + d * 79) * 1.5;

        point(x + width / 2, y + height / 2);
    }
}

function windowResized() {
    resizeCanvas(windowWidth, windowHeight);
}
