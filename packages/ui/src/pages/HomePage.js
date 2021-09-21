import React from 'react';
import { 
    Hero,
    MissionSlide
} from 'components';
import { useTheme } from '@emotion/react';

function HomePage() {
    const theme = useTheme();
    return (
        <div>
            <Hero text="Your portal to idea monetization" />
            <MissionSlide />
        </div >
    );
}

HomePage.propTypes = {
    
}

export { HomePage };