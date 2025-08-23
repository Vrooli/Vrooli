// Parametric box with lid
width = 40;
height = 30;
depth = 20;
wall = 2;

// Main box
difference() {
    cube([width, depth, height]);
    translate([wall, wall, wall])
        cube([width-2*wall, depth-2*wall, height]);
}
