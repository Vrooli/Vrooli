import React from 'react';
import { 
    FeaturedProducts,
    Hero 
} from 'components';

function HomePage() {
    return (
        <div>
            <Hero text="Need a fast, large, or cheap spaceship?" subtext="We've got you covered" />
            <FeaturedProducts />
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };