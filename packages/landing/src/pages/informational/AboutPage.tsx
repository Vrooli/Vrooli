import { useState, useEffect } from 'react';
import { makeStyles } from '@material-ui/styles';
import { pageStyles } from '../styles';
import { combineStyles } from 'utils';
import MattSketch from 'assets/img/thought-sketch-edited-3.png';
import { Slide } from 'components';

const componentStyles = () => ({
    social: {
        width: '80px',
        height: '80px',
    }
})

const useStyles = makeStyles(combineStyles(pageStyles, componentStyles));

export const AboutPage = () => {
    const classes = useStyles();
   // Load all images on page
   const [width, setWidth] = useState(window.innerWidth);

   // Auto-updates width. Used to determine what size image to fetch
   useEffect(() => {
       const onResize = () => setWidth(window.innerWidth);
       window.addEventListener('resize', onResize);
       return () => {
           window.removeEventListener('resize', onResize);
       }
   })

   // Slides in order
   const slides = [
       {
           id: 'about-the-team',
           title: { text: 'About the Team', position: 'center' },
           background: { background: '#d7deef' },
           body: [
               {
                   content: [
                       { title: { text: 'Leader/Developer - Matt Halloran', variant: 'h5' } },
                   ]
               },
               {
                   content: [
                       { title: { text: `Matt is a 23-year-old with a life-long passion for contemplating the future. He is very detail-oriented, and spends a lot of time (arguably too much) examining our best chances for surviving the Great Filter. These thoughts led him to start Vrooli, in hopes that the platform will accelerate humanity's rate of progress - while preventing inequality from getting out of control.`, variant: 'h5' } },
                   ]
               },
               {
                   sm: 6,
                   content: [
                       { list: { variant: 'h5', items: [
                           { text: 'Bachelors in Computer Science, 3.85 GPA' }, 
                           { text: 'Plutus Pioneers 1st Cohort graduate'},
                           { text: 'Futurist🔭 , libertarian🗽,  vegan🌱' }
                       ]}}
                   ]
               },
               {
                   sm: 6,
                   content: [{ image: { src: MattSketch, alt: 'Illustrated drawing of the founder of Vrooli - Matt Halloran' } }]
               },
               {
                   buttons: [
                       { text: 'Other Projects', position: 'center', color: 'secondary', link: 'https://github.com/MattHalloran' },
                       { text: 'Contact/Support', position: 'center', color: 'secondary', link: 'https://matthalloran.info/' }
                   ]
               },
               {
                   content: [{ title: { text: 'Other Teammates - Possibly YOU!', variant: 'h5' } },]
               },
               {
                   content: [{ title: { text: `Vrooli has a bright future ahead, but we need all hands on deck! If you are interested in being an open-source contributor or team member, don't hesitate to reach out!`, variant: 'h5' } }]
               },
           ]
       },
   ]

   return (
       <div className={classes.root}>
           {slides.map((data, index) => <Slide key={`slide-${index}`} width={width} data={data} />)}
       </div >
   );
}